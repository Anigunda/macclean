package cleaner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the user configuration loaded from ~/.config/macclean/config.yaml.
// Everything is optional; zero values get defaults.
type Config struct {
	Keep           []string      `yaml:"keep"`            // substrings: matching items are suggested KEEP
	Delete         []string      `yaml:"delete"`          // substrings: force DELETE (overrides built-in keeps)
	QuarantineDays int           `yaml:"quarantine_days"` // auto-purge quarantine batches after N days (default 7)
	StaleDays      int           `yaml:"stale_days"`      // node_modules untouched this long are cleanable (default 30)
	ExtraTargets   []ExtraTarget `yaml:"extra_targets"`   // custom categories
}

// ExtraTarget is a user-defined cleanable category (its paths' *contents* are cleaned).
type ExtraTarget struct {
	ID    string   `yaml:"id"`
	Name  string   `yaml:"name"`
	Paths []string `yaml:"paths"`
}

// userConfig holds the loaded config (nil = no file / defaults). Set via LoadConfig.
var userConfig *Config

// DefaultQuarantineDays / DefaultStaleDays are used when the config does not set them.
const (
	DefaultQuarantineDays = 7
	DefaultStaleDays      = 30
)

// ConfigPath is where the config file lives: ~/.config/macclean/config.yaml
// (pinned to the conventional unix path rather than macOS Application Support).
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "macclean", "config.yaml")
}

// LoadConfig reads the config file (missing file is fine - defaults apply).
// It must be called before Targets()/Suggest() are used.
func LoadConfig() error {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			userConfig = &Config{}
			return nil
		}
		return err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	userConfig = &cfg
	return nil
}

// GetConfig returns the loaded config (defaults if LoadConfig was never called).
func GetConfig() *Config {
	if userConfig == nil {
		return &Config{}
	}
	return userConfig
}

// QuarantineDays returns the retention for quarantine batches.
func (c *Config) QuarantineDaysOrDefault() int {
	if c.QuarantineDays > 0 {
		return c.QuarantineDays
	}
	return DefaultQuarantineDays
}

// StaleDaysOrDefault returns the age threshold for stale node_modules.
func (c *Config) StaleDaysOrDefault() int {
	if c.StaleDays > 0 {
		return c.StaleDays
	}
	return DefaultStaleDays
}

// MatchesKeep reports whether a name matches a user keep rule.
func (c *Config) MatchesKeep(name string) bool {
	return containsFold(name, c.Keep)
}

// MatchesDelete reports whether a name matches a user delete rule.
func (c *Config) MatchesDelete(name string) bool {
	return containsFold(name, c.Delete)
}

func containsFold(name string, patterns []string) bool {
	lower := strings.ToLower(name)
	for _, p := range patterns {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" && strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// SampleConfig is written by 'macclean config --init'.
const SampleConfig = `# macclean configuration
# Docs: item "names" are the things you see in 'macclean scan --detail' / the TUI.

# Substrings: items containing one of these are suggested KEEP (unchecked in the TUI).
keep:
  - ollama
  - huggingface

# Substrings: force DELETE - useful to override a built-in keep rule.
delete: []

# Days a quarantine batch is kept before it is auto-purged.
quarantine_days: 7

# A project node_modules untouched for this many days is considered stale/cleanable.
stale_days: 30

# Your own categories. The CONTENTS of each path are cleaned (never the dir itself).
extra_targets: []
#  - id: my-cache
#    name: My Cache
#    paths:
#      - ~/work/.build-cache
`

// WriteSampleConfig creates the config file with commented defaults.
// Refuses to overwrite an existing file.
func WriteSampleConfig() (string, error) {
	path := ConfigPath()
	if _, err := os.Stat(path); err == nil {
		return path, fmt.Errorf("config already exists: %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, err
	}
	if err := os.WriteFile(path, []byte(SampleConfig), 0o644); err != nil {
		return path, err
	}
	return path, nil
}
