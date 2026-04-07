package cmd

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ZiplEix/stew/v2/internal/tracker"
	"github.com/ZiplEix/stew/v2/stewlang"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var watchMode bool

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile .stew files to Go natively",
	Run: func(cmd *cobra.Command, args []string) {
		moduleBase := getModuleBase()
		t := tracker.NewTracker()

		// Full Scan Phase
		fmt.Println("🍲 Stew Compiler starting...")
		err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			if !d.IsDir() && strings.HasSuffix(path, ".stew") {
				compileStewFile(path, moduleBase, t)
			}
			return nil
		})

		if err != nil {
			log.Fatalf("Error scanning directory: %v", err)
		}

		if err := t.Save(); err != nil {
			log.Printf("Warning: could not save tracker: %v", err)
		}

		if watchMode {
			startCompileWatcher(moduleBase)
		}
	},
}

func getModuleBase() string {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		log.Fatalf("Error reading go.mod: please run stew compile at the root of the Go project. (%v)", err)
	}
	lines := strings.SplitSeq(string(content), "\n")
	for line := range lines {
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

func getPackageNameSafe(sourcePath string) string {
	dir := filepath.Dir(sourcePath)
	if dir == "." {
		return "main"
	}
	base := filepath.Base(dir)
	// Sanitize package name: Go packages cannot contain hyphens
	return strings.ReplaceAll(base, "-", "_")
}

func resolveOutputName(sourcePath string) string {
	dir := filepath.Dir(sourcePath)
	base := filepath.Base(sourcePath)

	if base == "@page.stew" {
		return filepath.Join(dir, "stew.page.go")
	}
	if base == "@layout.stew" {
		return filepath.Join(dir, "stew.layout.go")
	}

	// Component matching mappings MonComposant.stew -> MonComposant.go
	baseName := strings.TrimSuffix(base, ".stew")
	return filepath.Join(dir, baseName+".go")
}

func compileStewFile(path string, moduleBase string, t *tracker.Tracker) {
	if !strings.HasSuffix(path, ".stew") {
		return // Strict enforcement mapping rules
	}

	content, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Error reading %s: %v\n", path, err)
		return
	}

	baseName := filepath.Base(path)
	funcName := strings.TrimSuffix(baseName, ".stew")
	switch baseName {
	case "@page.stew":
		funcName = "Page"
	case "@layout.stew":
		funcName = "Layout"
	}

	pkgName := getPackageNameSafe(path)

	outputContent, extraArtifacts, err := stewlang.Compile(funcName, pkgName, moduleBase, path, string(content))
	if err != nil {
		log.Printf("Compile error in %s: %v\n", path, err)
		return
	}

	outFile := resolveOutputName(path)
	err = os.WriteFile(outFile, []byte(outputContent), 0644)
	if err != nil {
		log.Printf("Error writing output for %s: %v\n", path, err)
	} else {
		log.Printf("Compiled: %s -> %s\n", path, outFile)
		t.Add(outFile)
		for _, artifact := range extraArtifacts {
			t.Add(artifact)
		}
	}
}

func startCompileWatcher(moduleBase string) {
	t := tracker.NewTracker()
	fmt.Println("👀 Watching for changes (.stew files)...")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Add all subdirectories recursively
	err = filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && d.Name() != ".git" && d.Name() != "node_modules" {
			return watcher.Add(path)
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	// 100ms Debouncing loop
	changedFiles := make(map[string]bool)
	deletedFiles := make(map[string]bool)
	var timer *time.Timer

	processChanges := func() {
		for f := range changedFiles {
			if deletedFiles[f] {
				outFile := resolveOutputName(f)
				err := os.Remove(outFile)
				if err == nil || os.IsNotExist(err) {
					log.Printf("Deleted strictly mapped file: %s (source %s removed)\n", outFile, f)
					t.Remove(outFile)
				}
			} else {
				compileStewFile(f, moduleBase, t)
			}
		}
		_ = t.Save()
		changedFiles = make(map[string]bool)
		deletedFiles = make(map[string]bool)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Dynamic folder watch addition
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					watcher.Add(event.Name)
				}
			}

			if !strings.HasSuffix(event.Name, ".stew") {
				continue // Skip processing non-.stew elements strict mapping exclusion.
			}

			changedFiles[event.Name] = true
			if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
				deletedFiles[event.Name] = true
			}

			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(100*time.Millisecond, processChanges)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

func init() {
	rootCmd.AddCommand(compileCmd)
	compileCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch for changes on .stew files and trigger debounce auto-reload bindings.")
}
