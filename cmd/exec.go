package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ZiplEix/stew/v2/internal/config"
	"github.com/ZiplEix/stew/v2/internal/utils"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec [command]",
	Short: "Execute a one-shot command with .env support",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := utils.LoadConfig(".stew.yaml")
		if err != nil {
			utils.LoadEnvFiles(&config.Config{})
		} else {
			utils.LoadEnvFiles(cfg)
		}

		fullCommand := strings.Join(args, " ")

		color := "\033[37m"
		if cfg != nil && len(cfg.Colors) > 0 {
			color = cfg.Colors[0]
		}

		script := config.Script{
			Name: "exec",
			Run:  fullCommand,
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		fmt.Printf("⚡ Executing one-shot: %s\n", fullCommand)
		if err := executeAndPrefix(ctx, script, color); err != nil {
			fmt.Printf("\n%s[ERROR]%s Execution failed: %v\n", errorColor, resetColor, err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
