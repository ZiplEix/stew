package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZiplEix/stew/v2/internal/tracker"
	"github.com/ZiplEix/stew/v2/internal/utils"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove build artifacts and generated files recursivelly",
	Long:  `Scans the project based on patterns defined in .stew.yaml and removes files and directories. Supports recursive file matching with "**/*" pattern.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🧹 Cleaning up project...")

		// Phase 1: always clean stew-tracked compiler artifacts
		t := tracker.NewTracker()
		t.CleanAll()

		// Phase 2: clean patterns from .stew.yaml (optional)
		cfg, err := utils.LoadConfig(".stew.yaml")
		if err != nil {
			fmt.Printf("%s[ERROR]%s Could not load configuration: %v\n", errorColor, resetColor, err)
			os.Exit(1)
		}

		for _, pattern := range cfg.Clean {
			if strings.HasPrefix(pattern, "**/*") {
				fileSuffix := strings.TrimPrefix(pattern, "**/*")
				cleanRecursively(fileSuffix)
			} else {
				cleanStandardPattern(pattern)
			}
		}

		fmt.Println("\n✨ Project is clean and fresh!")
	},
}

// cleanStandardPattern handles explicit files or directories like "bin/" or "tmp/main"
func cleanStandardPattern(pattern string) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Printf("⚠️  Invalid pattern '%s': %v\n", pattern, err)
		return
	}

	for _, match := range matches {
		removePath(match)
	}
}

func cleanRecursively(suffix string) {
	if suffix == "" {
		fmt.Println("⚠️  Recursive clean pattern is too broad (no suffix). Skipping.")
		return
	}

	fmt.Printf("🔍 Scanning recursively for files ending with '%s'...\n", suffix)

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("⚠️  Error accessing path %s: %v\n", path, err)
			return nil
		}

		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "bin") {
			return fs.SkipDir
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), suffix) {
			removePath(path)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("❌ Recursive scan failed: %v\n", err)
	}
}

// removePath abstractly handles file/directory removal and logging
func removePath(path string) {
	if path == "." || path == ".." || path == ".git" || path == "go.mod" || path == ".stew.yaml" {
		fmt.Printf("⚠️  Blocked removal of critical path: %s\n", path)
		return
	}

	err := os.RemoveAll(path)
	if err != nil {
		fmt.Printf("❌ Failed to remove %s: %v\n", path, err)
	} else {
		icon := "🗑️ "
		fileInfo, _ := os.Stat(path)
		if fileInfo != nil && fileInfo.IsDir() {
			icon = "📂"
		}
		fmt.Printf("%s Removed: %s\n", icon, path)
	}
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
