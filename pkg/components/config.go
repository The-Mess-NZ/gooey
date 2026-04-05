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

// TouchConfig describes touch device selection and calibration.
// TODO: IsLandscape isn't really implemented yet in terms of drawing the UI differently.
type TouchConfig struct {
	DevicePath    string `json:"devicePath"`
	MinXRaw       int32  `json:"minXRaw"`
	MaxXRaw       int32  `json:"maxXRaw"`
	MinYRaw       int32  `json:"minYRaw"`
	MaxYRaw       int32  `json:"maxYRaw"`
	ScreenXPixels int    `json:"ScreenXPixels"`
	ScreenYPixels int    `json:"ScreenYPixels"`
	InvertX       bool   `json:"invertX"`
	InvertY       bool   `json:"invertY"`
	IsLandscape   bool   `json:"isLandscape"`
}

// Config contains component defaults and runtime device settings loaded from disk.
type Config struct {
	Components map[string]ComponentDefaults `json:"components"`
	Touch      TouchConfig                  `json:"touch,omitempty"`
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

// SaveConfig writes the currently loaded configuration back to disk.
func SaveConfig(path string) error {
	configMu.RLock()
	data, err := json.MarshalIndent(currentConfig, "", "  ")
	configMu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal component config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write component config: %w", err)
	}
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
		Touch: TouchConfig{
			DevicePath:    "/dev/null",
			MinXRaw:       0,
			MaxXRaw:       0,
			MinYRaw:       0,
			MaxYRaw:       0,
			ScreenXPixels: 0,
			ScreenYPixels: 0,
			InvertX:       false,
			InvertY:       false,
			IsLandscape:   false,
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
	merged.Touch = mergeTouchConfig(base.Touch, override.Touch)
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

// TouchSettings returns the active touch device and calibration settings.
func TouchSettings() TouchConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	return currentConfig.Touch
}

// SetTouchSettings updates the active touch device and calibration settings.
func SetTouchSettings(cfg TouchConfig) {
	configMu.Lock()
	currentConfig.Touch = mergeTouchConfig(defaultConfig().Touch, cfg)
	configMu.Unlock()
}

func mergeTouchConfig(base, override TouchConfig) TouchConfig {
	if override.DevicePath != "" {
		base.DevicePath = override.DevicePath
	}
	if override.MinXRaw != 0 {
		base.MinXRaw = override.MinXRaw
	}
	if override.MaxXRaw != 0 {
		base.MaxXRaw = override.MaxXRaw
	}
	if override.MinYRaw != 0 {
		base.MinYRaw = override.MinYRaw
	}
	if override.MaxYRaw != 0 {
		base.MaxYRaw = override.MaxYRaw
	}
	if override.ScreenXPixels != 0 {
		base.ScreenXPixels = override.ScreenXPixels
	}
	if override.ScreenYPixels != 0 {
		base.ScreenYPixels = override.ScreenYPixels
	}
	base.InvertX = override.InvertX
	base.InvertY = override.InvertY
	base.IsLandscape = override.IsLandscape
	return base
}
