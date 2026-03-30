package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const fileName = ".stew.yaml"
const layoutFileName = "layout.templ"

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

const defaultLayout = `package main

import (
	"github.com/ZiplEix/stew/sdk/live"
	"os"
)

templ Layout(title string) {
	<!DOCTYPE html>
	<html lang="fr">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ title }</title>
			
			<script src="https://unpkg.com/htmx.org@1.9.10"></script>
			<script src="https://unpkg.com/idiomorph/dist/idiomorph-ext.min.js"></script>
		</head>
		<body hx-ext="morph">
			{ children... }

			if os.Getenv("STEW_DEV") == "true" {
				@templ.Raw(live.InjectScript())
			}
		</body>
	</html>
}
`

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Init a new configuration file .stew.yaml",
	Long: `Create a new .stew.yaml file in the curent directory.
This file is used to define personalized script for the development workflow.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🍲 Preparing your Stew project...")
		fmt.Println("---------------------------------")

		handleFileCreation(fileName, defaultConfig)
		handleFileCreation(layoutFileName, defaultLayout)

		fmt.Println("\n✅ Setup complete! Run 'stew install' to get your tools.")
	},
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
		fmt.Printf("✨ %s created/updated\n", path)
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
