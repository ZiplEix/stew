package config

type Config struct {
	Commands map[string]CommandConfig `yaml:"commands"`
	Colors   []string                 `yaml:"colors"`
	Requires []Dependency             `yaml:"requires"`
	Clean    []string                 `yaml:"clean"`
	EnvFiles []string                 `yaml:"env_files"`
}

type Dependency struct {
	Name    string `yaml:"name"`
	Package string `yaml:"package"`
}

type Script struct {
	Name  string `yaml:"name"`
	Run   string `yaml:"run"`
	Watch bool   `yaml:"watch"`
}

type CommandConfig struct {
	Parallel bool     `yaml:"parallel"`
	Scripts  []Script `yaml:"scripts"`
}
