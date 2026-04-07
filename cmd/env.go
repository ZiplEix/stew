package cmd

import (
	"fmt"
	"os"

	"github.com/ZiplEix/stew/v2/internal/utils"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "List all environment variables loaded from .env",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := utils.LoadConfig(".stew.yaml")
		if err != nil {
			fmt.Printf("%s[ERROR]%s Could not load configuration: %v\n", errorColor, resetColor, err)
			os.Exit(1)
		}

		utils.LoadEnvFiles(config)
		env, _ := godotenv.Read()
		if len(env) == 0 {
			fmt.Println("Empty .env or no .env file found.")
			return
		}
		fmt.Println("Current .env variables:")
		for k, v := range env {
			fmt.Printf("  %s=%s\n", k, v)
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
