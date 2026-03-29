package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "v1.0.0"
	BuildTime = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Stew",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🍲 Stew %s\n", Version)
		fmt.Printf("📅 Build time: %s\n", BuildTime)
		fmt.Println("🔗 Repository: https://github.com/ZiplEix/stew")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
