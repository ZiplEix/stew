package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const fileName = ".stew.yaml"
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

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Init a new configuration file .stew.yaml",
	Long: `Create a new .stew.yaml file in the curent directory.
This file is used to define personalized script for the development workflow.`,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(fileName); err == nil {
			fmt.Printf("⚠️  %s already exists\n", fileName)
			return
		}

		err := os.WriteFile(fileName, []byte(defaultConfig), 0644)
		if err != nil {
			fmt.Printf("❌ Error while creating %s: %s\n", fileName, err.Error())
			return
		}

		fmt.Printf("✅ %s created successfully\n", fileName)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
