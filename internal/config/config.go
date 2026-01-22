package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all gitty configuration
type Config struct {
	Git    GitConfig    `yaml:"git"`
	AI     AIConfig     `yaml:"ai"`
	UI     UIConfig     `yaml:"ui"`
	GitHub GitHubConfig `yaml:"github"`
}

// GitConfig holds git-related settings
type GitConfig struct {
	UserName  string `yaml:"user_name"`
	UserEmail string `yaml:"user_email"`
	Editor    string `yaml:"editor"`
}

// AIConfig holds AI commit settings
type AIConfig struct {
	Provider    string  `yaml:"provider"` // openai, anthropic
	Model       string  `yaml:"model"`
	APIKey      string  `yaml:"api_key"`
	MaxDiffSize int     `yaml:"max_diff_size"`
	Temperature float64 `yaml:"temperature"`
}

// UIConfig holds UI preferences
type UIConfig struct {
	Theme       string `yaml:"theme"` // charm, dracula, catppuccin
	ShowIcons   bool   `yaml:"show_icons"`
	AnimationMs int    `yaml:"animation_ms"`
}

// GitHubConfig holds GitHub publishing settings
type GitHubConfig struct {
	DefaultVisibility string `yaml:"default_visibility"` // public, private
	NormalizeAuthor   bool   `yaml:"normalize_author"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Git: GitConfig{
			UserName:  "",
			UserEmail: "",
			Editor:    "vim",
		},
		AI: AIConfig{
			Provider:    "openai",
			Model:       "gpt-4o-mini",
			APIKey:      "",
			MaxDiffSize: 4000,
			Temperature: 0.7,
		},
		UI: UIConfig{
			Theme:       "charm",
			ShowIcons:   true,
			AnimationMs: 100,
		},
		GitHub: GitHubConfig{
			DefaultVisibility: "public",
			NormalizeAuthor:   false,
		},
	}
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gitty.yaml"
	}
	return filepath.Join(home, ".config", "gitty", "config.yaml")
}

// Load loads the configuration from file or returns default
func Load() (*Config, error) {
	cfg := DefaultConfig()

	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create default config
			_ = Save(cfg)
			return cfg, nil
		}
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return DefaultConfig(), err
	}

	// Override API key from environment if not set in config
	if cfg.AI.APIKey == "" {
		cfg.AI.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	return cfg, nil
}

// Save saves the configuration to file
func Save(cfg *Config) error {
	path := ConfigPath()

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// EnsureConfig ensures the config file exists with defaults
func EnsureConfig() (*Config, error) {
	cfg, err := Load()
	if err != nil {
		cfg = DefaultConfig()
		if saveErr := Save(cfg); saveErr != nil {
			return cfg, saveErr
		}
	}
	return cfg, nil
}
