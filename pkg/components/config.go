package components

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

// ComponentDefaults describes default styling loaded from config.json.
type ComponentDefaults struct {
	FontSize    float64 `json:"fontSize,omitempty"`
	TextPadding int     `json:"textPadding,omitempty"`
	BorderWidth int     `json:"borderWidth,omitempty"`
}

// Config contains component defaults loaded from disk.
type Config struct {
	Components map[string]ComponentDefaults `json:"components"`
}

var (
	configMu      sync.RWMutex
	currentConfig = defaultConfig()
)

// LoadConfig loads the Gooey component defaults from a JSON file.
func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("unmarshal component config: %w", err)
	}

	configMu.Lock()
	currentConfig = mergeConfig(defaultConfig(), cfg)
	configMu.Unlock()
	return nil
}

// LoadConfigFromCandidates loads the first existing config file from the provided candidates.
func LoadConfigFromCandidates(paths ...string) (string, error) {
	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := LoadConfig(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", err
		}
		return path, nil
	}

	return "", nil
}

func defaultConfig() Config {
	return Config{
		Components: map[string]ComponentDefaults{
			NodeTypeContainer: {},
			NodeTypeLabel: {
				FontSize:    13,
				TextPadding: 4,
			},
			NodeTypeButton: {
				FontSize:    14,
				TextPadding: 6,
				BorderWidth: 2,
			},
		},
	}
}

func mergeConfig(base, override Config) Config {
	merged := Config{Components: make(map[string]ComponentDefaults, len(base.Components))}
	for key, value := range base.Components {
		merged.Components[key] = value
	}
	for key, value := range override.Components {
		merged.Components[key] = mergeComponentDefaults(merged.Components[key], value)
	}
	return merged
}

func mergeComponentDefaults(base, override ComponentDefaults) ComponentDefaults {
	if override.FontSize != 0 {
		base.FontSize = override.FontSize
	}
	if override.TextPadding != 0 {
		base.TextPadding = override.TextPadding
	}
	if override.BorderWidth != 0 {
		base.BorderWidth = override.BorderWidth
	}
	return base
}

func componentDefaults(nodeType string) ComponentDefaults {
	configMu.RLock()
	defer configMu.RUnlock()

	if defaults, ok := currentConfig.Components[nodeType]; ok {
		return defaults
	}
	return ComponentDefaults{}
}
