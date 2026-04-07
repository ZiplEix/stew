package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Module struct {
	Path string
	Dir  string
	Main bool
}

// ExtractVendorAssets scans the project's Go modules dependencies.
// If a dependency has a 'static/' folder, it copies its contents to 'static/vendor/<module_name>/'.
func ExtractVendorAssets() {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("⚠️  Warning: Unable to list Go modules to extract vendor assets: %v\n", err)
		return
	}

	// Output contains multiple concatenated JSON objects. We need to split or decode stream.
	decoder := json.NewDecoder(strings.NewReader(string(out)))
	
	for {
		var mod Module
		err := decoder.Decode(&mod)
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip errors
		}

		if mod.Main || mod.Dir == "" {
			continue
		}

		staticSrc := filepath.Join(mod.Dir, "static")
		if _, err := os.Stat(staticSrc); os.IsNotExist(err) {
			continue
		}

		// Use the last part of module path or the whole path?
		// Better to use the whole path to avoid collisions, e.g., static/vendor/github.com/user/lib
		vendorDest := filepath.Join("static", "vendor", mod.Path)
		
		err = copyDir(staticSrc, vendorDest)
		if err != nil {
			fmt.Printf("❌ Failed to copy assets from module %s: %v\n", mod.Path, err)
		} else {
			fmt.Printf("📦 Extracted assets from %s to %s\n", mod.Path, vendorDest)
		}
	}
}

// copyDir recursively copies a directory tree.
func copyDir(src string, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile copies a single file.
func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, info.Mode())
}
