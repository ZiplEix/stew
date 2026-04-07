package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ZiplEix/stew/internal/utils"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install all required Go binaries defined in .stew.yaml",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := utils.LoadConfig(".stew.yaml")
		if err != nil {
			fmt.Printf("%s[ERROR]%s Could not load configuration: %v\n", errorColor, resetColor, err)
			os.Exit(1)
		}

		if len(config.Requires) == 0 {
			fmt.Println("✨ No dependencies defined in 'requires' section.")
			return
		}

		fmt.Println("📦 Installing dependencies...")

		// Ensure wasm_exec.js is present for client-side features
		utils.EnsureWasmRuntime()

		for _, dep := range config.Requires {
			if _, err := exec.LookPath(dep.Name); err == nil {
				fmt.Printf("✅ %-15s is already installed. Skipping.\n", dep.Name)
				continue
			}

			fmt.Printf("⏳ Installing %s (%s)...\n", dep.Name, dep.Package)

			// Execute: go install <package>
			installArgs := []string{"install", dep.Package}
			runCmd := exec.Command("go", installArgs...)
			runCmd.Stdout = os.Stdout
			runCmd.Stderr = os.Stderr

			if err := runCmd.Run(); err != nil {
				fmt.Printf("❌ Failed to install %s: %v\n", dep.Name, err)
			} else {
				fmt.Printf("✅ %s installed successfully!\n", dep.Name)
			}
		}

		fmt.Println("\n🚀 All dependencies are ready.")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
