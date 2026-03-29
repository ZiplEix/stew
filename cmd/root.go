package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "stew",
	Short: "🍲 A minimalist task runner and orchestrator for Go projects",
	Long: `Stew is a lightweight CLI tool designed for Go developers who want a 
modern developer experience without the need for Node.js or Bun.

It allows you to:
- Define and run custom scripts (Serial or Parallel)
- Manage environment variables automatically via .env files
- Install Go-based dependencies (like templ or air)
- Clean up project artifacts recursively
- Check your development environment in one command

Perfect for Go + Templ + HTMX stacks.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}
