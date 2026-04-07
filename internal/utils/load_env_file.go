package utils

import (
	"fmt"
	"os"

	"github.com/ZiplEix/stew/v2/internal/config"
	"github.com/joho/godotenv"
)

const resetColor = "\033[0m"

func LoadEnvFiles(cfg *config.Config) {
	files := cfg.EnvFiles
	if len(files) == 0 {
		if _, err := os.Stat(".env"); err == nil {
			files = []string{".env"}
		}
	}

	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			err := godotenv.Load(file)
			if err != nil {
				fmt.Printf("%s[WARN]%s Error loading %s: %v\n", "\033[33m", resetColor, file, err)
			} else {
				fmt.Printf("🌱 Environment variables loaded from %s\n", file)
			}
		}
	}
}
