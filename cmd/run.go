package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/ZiplEix/stew/internal/config"
	"github.com/ZiplEix/stew/internal/utils"
	"github.com/spf13/cobra"
)

const (
	resetColor = "\033[0m"
	errorColor = "\033[31m"
)

var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Run a command defined in .stew.yaml",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config, err := utils.LoadConfig(".stew.yaml")
		if err != nil {
			fmt.Printf("%s[ERROR]%s Could not load configuration: %v\n", errorColor, resetColor, err)
			os.Exit(1)
		}

		utils.LoadEnvFiles(config)

		if len(args) == 0 {
			fmt.Println("🍲 Available stew recipes (commands):")

			keys := make([]string, 0, len(config.Commands))
			for k := range config.Commands {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, name := range keys {
				c := config.Commands[name]
				mode := "Serial"
				if c.Parallel {
					mode = "Parallel"
				}
				fmt.Printf("  - %-12s (%d scripts, %s mode)\n", name, len(c.Scripts), mode)
			}
			return
		}

		target := args[0]
		cmdConfig, ok := config.Commands[target]
		if !ok {
			fmt.Printf("%s[ERROR]%s Command '%s' not found in .stew.yaml\n", errorColor, resetColor, target)

			fmt.Println("\nAvailable commands:")
			for name := range config.Commands {
				fmt.Printf("  - %s\n", name)
			}
			os.Exit(1)
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		shouldWatch := false
		for _, s := range cmdConfig.Scripts {
			if s.Watch {
				shouldWatch = true
				break
			}
		}

		if shouldWatch {
			go utils.StartWatcher(ctx)
		}

		if cmdConfig.Parallel {
			runParallel(ctx, config.Colors, cmdConfig.Scripts)
		} else {
			runSerial(ctx, config.Colors, cmdConfig.Scripts)
		}

		fmt.Println("\n🍲 Stew finished cooking. See you next time!")
	},
}

func runParallel(ctx context.Context, colors []string, scripts []config.Script) {
	fmt.Println("🍲 Stew is simmering (Parallel Mode)...")
	var wg sync.WaitGroup
	for i, s := range scripts {
		wg.Add(1)
		color := colors[i%len(colors)]
		go func(script config.Script, clr string) {
			defer wg.Done()
			if err := executeAndPrefix(ctx, script, clr); err != nil {
				if ctx.Err() == nil {
					fmt.Printf("\n%s[ERROR]%s Command '%s' failed: %v\n", errorColor, resetColor, script.Name, err)
					os.Exit(1)
				}
			}
		}(s, color)
	}
	wg.Wait()
}

func runSerial(ctx context.Context, colors []string, scripts []config.Script) {
	fmt.Println("🍲 Stew is cooking (Serial Mode)...")
	for i, s := range scripts {
		if ctx.Err() != nil {
			break
		}
		color := colors[i%len(colors)]

		if err := executeAndPrefix(ctx, s, color); err != nil {
			if ctx.Err() == nil {
				fmt.Printf("\n%s[ERROR]%s Command '%s' failed: %v\n", errorColor, resetColor, s.Name, err)
				os.Exit(1)
			}
		}
	}
}

func executeAndPrefix(ctx context.Context, script config.Script, color string) error {
	parts := strings.Fields(script.Run)
	if len(parts) == 0 {
		return nil
	}

	binaryName := parts[0]
	if _, err := exec.LookPath(binaryName); err != nil {
		return fmt.Errorf("executable '%s' not found in your PATH", binaryName)
	}

	cmd := exec.CommandContext(ctx, binaryName, parts[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Env = append(os.Environ(), "STEW_DEV=true", "FORCE_COLOR=true")

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	name := script.Name
	if len(name) > 10 {
		name = name[:10]
	}
	formattedColor := utils.FormatColor(color)
	prefix := fmt.Sprintf("%s[%-10s]%s ", formattedColor, name, resetColor)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}

	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		}
	}()

	prefixWriter := &utils.LinePrefixWriter{
		Prefix: prefix,
		Output: os.Stdout,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(prefixWriter, stdout)
	}()

	go func() {
		defer wg.Done()
		io.Copy(prefixWriter, stderr)
	}()

	err := cmd.Wait()
	wg.Wait()

	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("execution failed: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)
}
