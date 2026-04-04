package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Tracker struct {
	TrackedFiles []string `json:"tracked_files"`
}

const TrackerFile = ".stew/compiled.json"

// NewTracker loads the existing tracker or initializes a new one
func NewTracker() *Tracker {
	t := &Tracker{TrackedFiles: []string{}}
	
	if info, err := os.Stat(TrackerFile); err == nil && !info.IsDir() {
		data, err := os.ReadFile(TrackerFile)
		if err == nil {
			_ = json.Unmarshal(data, t)
		}
	}
	
	return t
}

// Add appends a file to the tracking list if not already present
func (t *Tracker) Add(filePath string) {
	// Deduplicate before adding
	for _, f := range t.TrackedFiles {
		if f == filePath {
			return
		}
	}
	t.TrackedFiles = append(t.TrackedFiles, filePath)
}

// Remove deregisters a file from the tracked list
func (t *Tracker) Remove(filePath string) {
	filtered := t.TrackedFiles[:0]
	for _, f := range t.TrackedFiles {
		if f != filePath {
			filtered = append(filtered, f)
		}
	}
	t.TrackedFiles = filtered
}

// Save serializes the tracked files into .stew/compiled.json
func (t *Tracker) Save() error {
	dir := filepath.Dir(TrackerFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(TrackerFile, data, 0644)
}

// CleanAll removes all tracked generated outputs, then purges the .stew/ directory.
func (t *Tracker) CleanAll() {
	if len(t.TrackedFiles) == 0 {
		fmt.Println("ℹ️  No tracked compiler artifacts found.")
	} else {
		fmt.Printf("🗑️  Removing %d tracked compiler artifacts...\n", len(t.TrackedFiles))
		for _, f := range t.TrackedFiles {
			if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
				fmt.Printf("⚠️  Failed to remove %s: %v\n", f, err)
			} else {
				fmt.Printf("🗆 Removed: %s\n", f)
			}
		}
	}

	// Always purge the .stew/ cache directory
	if err := os.RemoveAll(".stew"); err != nil {
		fmt.Printf("⚠️  Could not remove .stew/: %v\n", err)
	} else {
		fmt.Println("🗆 Removed: .stew/ (compiler cache)")
	}
}
