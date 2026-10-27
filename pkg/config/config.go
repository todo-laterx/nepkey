package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	InputMethod     string `yaml:"input_method"`      // "Telex 2", "Telex", "VNI", "VIQR"
	Charset         string `yaml:"charset"`           // "Unicode", "Unicode tổ hợp", "TCVN3 (ABC)", "VNI Windows"
	OutputMethod    string `yaml:"output_method"`     // "hex", "clipboard"
	DevicePath      string `yaml:"device_path"`       // "auto" or specific /dev/input/...
	UinputPath      string `yaml:"uinput_path"`       // default "/dev/uinput"
	ToggleKey       string `yaml:"toggle_key"`        // "Ctrl+Space", "Alt+Shift"
	Enabled         bool   `yaml:"enabled"`           // default active status
	FreeToneMarking bool   `yaml:"free_tone_marking"` // Bỏ dấu tự do
	ModernToneStyle bool   `yaml:"modern_tone_style"` // Đặt dấu kiểu mới (òa, òe, ùy)
	AutoCorrect     bool   `yaml:"auto_correct"`      // Tự động sửa lỗi chính tả
	MacroEnabled    bool   `yaml:"macro_enabled"`     // Bật tính năng gõ tắt
	MacroFile       string `yaml:"macro_file"`        // Đường dẫn file gõ tắt macros.yaml
	DebugLog        bool   `yaml:"debug_log"`         // Ghi log gỡ lỗi ra /tmp/nepkey-debug.log
}

// ResolveConfigPath locates config.yaml in user home ~/.config/nepkey or local path
func ResolveConfigPath(path string) string {
	if path != "" && path != "configs/config.yaml" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		userCfg := filepath.Join(home, ".config", "nepkey", "config.yaml")
		if _, err := os.Stat(userCfg); err == nil {
			return userCfg
		}
	}
	if _, err := os.Stat("configs/config.yaml"); err == nil {
		return "configs/config.yaml"
	}
	return path
}

// ResolveMacroPath locates macros.yaml in user home ~/.config/nepkey or local path
func ResolveMacroPath(path string) string {
	if path != "" && path != "configs/macros.yaml" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		userMacro := filepath.Join(home, ".config", "nepkey", "macros.yaml")
		if _, err := os.Stat(userMacro); err == nil {
			return userMacro
		}
	}
	if _, err := os.Stat("configs/macros.yaml"); err == nil {
		return "configs/macros.yaml"
	}
	return path
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		InputMethod:     "Telex 2",
		Charset:         "Unicode",
		OutputMethod:    "clipboard",
		DevicePath:      "auto",
		UinputPath:      "/dev/uinput",
		ToggleKey:       "Ctrl+Space",
		Enabled:         true,
		FreeToneMarking: true,
		ModernToneStyle: true,
		AutoCorrect:     true,
		MacroEnabled:    true,
		MacroFile:       ResolveMacroPath("configs/macros.yaml"),
		DebugLog:        false,
	}
}

// LoadConfig loads configuration from a YAML file, or returns default if not found
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()
	resolved := ResolveConfigPath(path)
	if resolved == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.MacroFile == "configs/macros.yaml" || cfg.MacroFile == "" {
		cfg.MacroFile = ResolveMacroPath(cfg.MacroFile)
	}

	return cfg, nil
}

// SaveConfig saves configuration to a YAML file
func (c *Config) SaveConfig(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// MacroTable maps abbreviations to their full replacement strings
type MacroTable map[string]string

// DefaultMacros returns a set of standard Vietnamese abbreviations
func DefaultMacros() MacroTable {
	return MacroTable{
		"vn":  "Việt Nam",
		"hn":  "Hà Nội",
		"hcm": "Hồ Chí Minh",
		"dc":  "được",
		"ko":  "không",
		"ntn": "như thế nào",
	}
}

// LoadMacros loads the macro table from a YAML file.
// If the file exists, it only loads the macros explicitly defined in the file.
// If the file does not exist, it falls back to DefaultMacros.
func LoadMacros(path string) (MacroTable, error) {
	resolved := ResolveMacroPath(path)
	if resolved == "" {
		return DefaultMacros(), nil
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultMacros(), nil
		}
		return nil, err
	}

	macros := make(MacroTable)
	if err := yaml.Unmarshal(data, &macros); err != nil {
		return nil, err
	}
	return macros, nil
}

// SaveMacros saves the macro table to a YAML file
func SaveMacros(path string, macros MacroTable) error {
	data, err := yaml.Marshal(macros)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
