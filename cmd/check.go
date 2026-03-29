package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/ZiplEix/stew/internal/utils"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if all required binaries are installed",
	Long:  `Scans your .stew.yaml file and verifies that every command's binary is available in your PATH.`,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := utils.LoadConfig(".stew.yaml")
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		binaries := make(map[string]bool)

		for _, cmdConfig := range config.Commands {
			for _, script := range cmdConfig.Scripts {
				parts := strings.Fields(script.Run)
				if len(parts) > 0 {
					binaries[parts[0]] = true
				}
			}
		}

		fmt.Println("🔍 Checking dependencies...")
		allOk := true

		for bin := range binaries {
			path, err := exec.LookPath(bin)
			if err != nil {
				fmt.Printf("❌ %-15s [NOT FOUND]\n", bin)
				allOk = false
			} else {
				fmt.Printf("✅ %-15s [FOUND at %s]\n", bin, path)
			}
		}

		if allOk {
			fmt.Println("\n✨ All dependencies are satisfied. Happy cooking!")
		} else {
			fmt.Println("\n⚠️  Some dependencies are missing. Please install them to use all stew commands.")
		}
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
