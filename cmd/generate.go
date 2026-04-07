package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ZiplEix/stew/v2/internal/generator"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var watch bool

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Automatically generate the router from the pages directory",
	Run: func(cmd *cobra.Command, args []string) {
		moduleName, err := generator.GetModuleName()
		if err != nil {
			fmt.Printf("❌ Error: Impossible to read go.mod : %v\n", err)
			os.Exit(1)
		}

		// Première exécution immédiate
		runGeneration(moduleName)

		if watch {
			startWatcher(moduleName)
		}
	},
}

func runGeneration(moduleName string) {
	scanner := generator.NewScanner("pages", moduleName)

	fmt.Println("🔍 Scanning pages...")
	tree, err := scanner.Scan()
	if err != nil {
		fmt.Printf("❌ Error: Impossible to scan pages : %v\n", err)
		return
	}

	writer := generator.NewWriter(tree)
	outputFile := "stew_router_gen.go"

	if err := writer.Generate(outputFile); err != nil {
		fmt.Printf("❌ Error: Impossible to generate router : %v\n", err)
		return
	}

	fmt.Printf("✅ [%s] Stew Router updated\n", time.Now().Format("15:04:05"))
}

func startWatcher(moduleName string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("❌ Error: Watcher init failed : %v\n", err)
		return
	}
	defer watcher.Close()

	watchRecursive := func(path string) {
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err == nil && info.IsDir() {
				watcher.Add(p)
			}
			return nil
		})
	}

	watchRecursive("pages")
	fmt.Println("👀 Watching for changes in /pages...")

	var timer *time.Timer
	const debounceDuration = 100 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if !event.Has(fsnotify.Chmod) {
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceDuration, func() {
					if event.Has(fsnotify.Create) {
						info, err := os.Stat(event.Name)
						if err == nil && info.IsDir() {
							watchRecursive(event.Name)
						}
					}
					runGeneration(moduleName)
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("⚠️  Watcher error: %v\n", err)
		}
	}
}

func init() {
	generateCmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for changes in the pages directory")
	rootCmd.AddCommand(generateCmd)
}
