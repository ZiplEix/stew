package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	stewConfigFile = ".stew.yaml"
	pagesDir       = "pages"
)

const defaultConfig = `commands:
  dev:
    parallel: true
    scripts:
      - name: templ
        run: templ generate --watch
        watch: true
      - name: app
        run: air
  build:
    parallel: false
    scripts:
      - name: templ
        run: templ generate
      - name: go
        run: go build -o ./bin/app .

env_files:
  - .env
  - .env.local

colors:
  - '\033[32m' # Green
  - '\033[34m' # Blue
  - '\033[35m' # Magenta
  - '\033[36m' # Cyan
  - '\033[33m' # Yellow
  - '\033[92m' # Light Green
  - '\033[94m' # Light Blue
  - '\033[95m' # Light Magenta

requires:
  - name: templ
    package: github.com/a-h/templ/cmd/templ@latest
  - name: air
    package: github.com/air-verse/air@latest
`

const mainGoContent = `package main

import (
	"fmt"
	"net/http"
	"os"
	"github.com/ZiplEix/stew/sdk/live"
)

func main() {
	mux := http.NewServeMux()

	// Register generated routes from /pages
	RegisterStewRoutes(mux)

	var handler http.Handler = mux
	if os.Getenv("STEW_DEV") == "true" {
		fmt.Println("🛠️  Development mode: Stew Middleware enabled")
		handler = live.Middleware(mux)
	}

	port := ":8080"
	fmt.Printf("🚀 Server ready at http://localhost%s\n", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
`

const rootLayoutContent = `package pages

import (
	"github.com/ZiplEix/stew/sdk/live"
	"os"
)

templ Layout(contents templ.Component) {
	<!DOCTYPE html>
	<html lang="fr">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>Stew App</title>
			<script src="https://unpkg.com/htmx.org@1.9.10"></script>
			<script src="https://unpkg.com/idiomorph/dist/idiomorph-ext.min.js"></script>
		</head>
		<body hx-ext="morph">
			@contents

			if os.Getenv("STEW_DEV") == "true" {
				@templ.Raw(live.InjectScript())
			}
		</body>
	</html>
}
`

const rootPageContent = `package pages

templ Page() {
	<div style="text-align: center; font-family: sans-serif; padding-top: 20vh;">
		<h1>🍲 Stew 2.0</h1>
		<p>Votre framework Go Fullstack est prêt.</p>
		<p>Modifiez <code>pages/stew.page.templ</code> pour commencer.</p>
	</div>
}
`

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [module-name]",
	Short: "Initialize a new Stew 2.0 project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🍲 Simmering a new Stew project...")
		fmt.Println("---------------------------------")

		// 1. Initialisation Go Mod
		handleGoMod(args)

		// 2. Création du fichier de config Stew
		handleFileCreation(stewConfigFile, defaultConfig)

		// 3. Création de l'arborescence /pages
		if err := os.MkdirAll(pagesDir, 0755); err != nil {
			fmt.Printf("❌ Error creating pages directory: %v\n", err)
		} else {
			fmt.Println("📂 pages/ directory created")
			handleFileCreation(filepath.Join(pagesDir, "stew.layout.templ"), rootLayoutContent)
			handleFileCreation(filepath.Join(pagesDir, "stew.page.templ"), rootPageContent)
		}

		// 4. Création du main.go
		handleFileCreation("main.go", mainGoContent)

		fmt.Println("\n🛠️  Executing post-init sequence...")

		// 5. stew install (Installe templ, air, etc.)
		runCommand("stew", "install")

		// 6. templ generate (Génère les fichiers _templ.go indispensables au routeur)
		runCommand("templ", "generate")

		// 7. stew generate (Génère le stew_router_gen.go)
		fmt.Println("🏗️  Generating router...")
		generateCmd.Run(generateCmd, []string{})

		// 8. go mod tidy (Nettoie les dépendances et résout les imports locaux)
		runCommand("go", "mod", "tidy")

		fmt.Println("\n✅ Project fully initialized and ready to run!")
		fmt.Println("👉 Run 'stew run dev' to start simmering.")
	},
}

func runCommand(name string, args ...string) {
	fmt.Printf("🏃 Running: %s %s...\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: %s %s failed: %v\n", name, args[0], err)
	}
}

func handleGoMod(args []string) {
	// Vérifier si go.mod existe déjà
	if _, err := os.Stat("go.mod"); err == nil {
		fmt.Println("ℹ️  go.mod already exists, skipping init.")
		return
	}

	// Si pas d'argument, on ne fait rien
	if len(args) == 0 {
		return
	}

	moduleName := args[0]
	if moduleName == "." {
		// Prendre le nom du dossier courant
		wd, _ := os.Getwd()
		moduleName = filepath.Base(wd)
	}

	fmt.Printf("📦 Running: go mod init %s\n", moduleName)
	cmd := exec.Command("go", "mod", "init", moduleName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Error during go mod init: %v\n", err)
	}
}

func handleFileCreation(path string, content string) {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("⚠️  %s already exists. Overwrite? (y/N): ", path)
		if !askConfirm() {
			fmt.Printf("   Skipped %s\n", path)
			return
		}
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error creating %s: %s\n", path, err)
	} else {
		fmt.Printf("✨ %s created\n", path)
	}
}

func askConfirm() bool {
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func init() {
	rootCmd.AddCommand(initCmd)
}
