package utils

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	errorColor       = "\033[31m"
	debounceDuration = 1000 * time.Millisecond
)

func TriggerReload() {
	client := http.Client{Timeout: debounceDuration}
	_, err := client.Get("http://localhost:9876/stew/trigger-reload")
	if err == nil {
		fmt.Println("🔄 Reload signal sent to browser")
	}
}

// StartWatcher monitors file changes and triggers the SDK reload
func StartWatcher(ctx context.Context) {
	pwd, _ := os.Getwd()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("%s[ERROR]%s Could not create watcher: %v\n", errorColor, resetColor, err)
		return
	}
	defer watcher.Close()

	err = filepath.WalkDir(pwd, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "bin" || d.Name() == "tmp") {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("%s[ERROR]%s Watcher failed to initialize: %v\n", errorColor, resetColor, err)
		return
	}

	fmt.Printf("👀 Stew is watching for changes in %s...\n", pwd)

	var timer *time.Timer

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				fmt.Printf("🔍 Change detected: %s\n", event.Name)
				if strings.HasPrefix(event.Name, ".") || strings.Contains(event.Name, ".stew.yaml") {
					continue
				}

				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceDuration, func() {
					TriggerReload()
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
