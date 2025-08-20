package utils

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/goccy/go-yaml"
)

// ConfigPath returns the path to the configuration file.
// It is located in the user's home directory.
func ConfigPath(fileName ...string) string {

	isWindows := runtime.GOOS == "windows"

	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Error getting user home directory", "err", err)
		panic(err)
	}
	if isWindows {
		return filepath.Join(home, "AppData", "Roaming", "gpt", filepath.Join(fileName...))
	} else {
		return filepath.Join(home, ".config", "gpt", filepath.Join(fileName...))
	}
}

// InitConfig initializes the configuration file if it does not exist.
func InitConfig(fileName string) error {
	if fileName == "" {
		fileName = ConfigPath("config.yaml")
	}
	baseDir := filepath.Dir(fileName)

	// Create directory if not exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		err := os.MkdirAll(baseDir, 0755)
		if err != nil {
			slog.Error("Error creating config directory", "err", err)
			return err
		}
	}
	// Create file if not exists
	if _, err := os.Stat(fileName); os.IsNotExist(err) {

		appConf := AppConf{
			LLM: LLM{
				Gateway:  "https://api.openai.com/v1/",
				ApiKey:   "your-api-key",
				Provider: "openai",
				Model:    "gpt-4o-mini",
			},
			LLMs: map[string]LLM{
				"openai": {
					Provider: "openai",
					Model:    "gpt-4o",
				},
				"deepseek": {
					Gateway:  "https://api.deepseek.ai/v1/",
					ApiKey:   "your-api-key",
					Provider: "deepseek",
					Model:    "deepseek-chat",
				},
			},
		}

		body, err := yaml.Marshal(&appConf)
		if err != nil {
			slog.Error("Error creating config file", "err", err)
			return err
		}

		err = os.WriteFile(fileName, body, 0644)
		if err != nil {
			slog.Error("Error writing config file", "err", err)
			return err
		}
	}
	return nil
}

// LoadConfig loads the configuration from the given file.
func LoadConfig(fileName string) (*AppConf, error) {
	if fileName == "" {
		fileName = ConfigPath("config.yaml")
	}
	body, err := os.ReadFile(fileName)
	if err != nil {
		slog.Error("Error reading config file", "err", err)
		return nil, err
	}
	appConf := AppConf{}
	err = yaml.Unmarshal(body, &appConf)
	if err != nil {
		slog.Error("Error parsing config file", "err", err)
		return nil, err
	}
	return &appConf, nil
}
