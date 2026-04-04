package main

import (
	"fmt"
	"io"
	"os"

	"github.com/ZiplEix/stew/stewlang"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <file.stew>")
		os.Exit(1)
	}

	fileName := os.Args[1]
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Determine page name from file
	baseName := fileName
	for i := len(fileName) - 1; i >= 0 && !os.IsPathSeparator(fileName[i]); i-- {
		baseName = fileName[i:]
	}
	if len(baseName) > 5 && baseName[len(baseName)-5:] == ".stew" {
		baseName = baseName[:len(baseName)-5]
	}

	output, err := stewlang.Compile(baseName, "pages", "github.com/ZiplEix/stew", fileName, string(content))
	if err != nil {
		fmt.Printf("Compilation Error:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== COMPILED OUTPUT ===")
	fmt.Println(output)
}
