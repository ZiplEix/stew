package utils

import (
	"fmt"
	"os"

	"github.com/ZiplEix/stew/v2/internal/config"
	"gopkg.in/yaml.v3"
)

func LoadConfig(path string) (*config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}

	var config config.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}
	return &config, nil
}
