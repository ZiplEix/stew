package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnsureWasmRuntime checks if wasm_exec.js exists in static/wasm.
// If it does not exist, it tries to copy it from the local TinyGo installation.
func EnsureWasmRuntime() {
	publicWasmDir := "static/wasm"
	wasmExecDest := filepath.Join(publicWasmDir, "wasm_exec.js")

	// Check if already exists
	if _, err := os.Stat(wasmExecDest); err == nil {
		fmt.Println("✅ WebAssembly runtime (wasm_exec.js) already present.")
		return
	}

	// Prepare directory
	if err := os.MkdirAll(publicWasmDir, 0755); err != nil {
		fmt.Printf("❌ Error creating public wasm directory: %v\n", err)
		return
	}

	fmt.Println("🚀 Fetching WebAssembly runtime from TinyGo...")
	out, err := exec.Command("tinygo", "env", "TINYGOROOT").Output()
	if err != nil {
		fmt.Println("⚠️  Warning: TinyGo is not installed. Wasm client features will be unavailable.")
		fmt.Println("   Install it: https://tinygo.org/")
		return
	}

	tinygoRoot := strings.TrimSpace(string(out))
	wasmExecSrc := filepath.Join(tinygoRoot, "targets", "wasm_exec.js")

	input, err := os.ReadFile(wasmExecSrc)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not read wasm_exec.js from TinyGo (%s)\n", wasmExecSrc)
		return
	}

	if err := os.WriteFile(wasmExecDest, input, 0644); err != nil {
		fmt.Printf("❌ Error: Could not write wasm_exec.js to %s: %v\n", wasmExecDest, err)
		return
	}

	fmt.Println("✨ WebAssembly runtime (wasm_exec.js) successfully copied!")
}
