package utils

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/goccy/go-yaml"
)

func DefaultConfigPath() string {

	isWindows := runtime.GOOS == "windows"

	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Error getting user home directory", "err", err)
		panic(err)
	}
	if isWindows {
		return filepath.Join(home, "AppData", "Roaming", "gpt", "config.yaml")
	} else {
		return filepath.Join(home, ".config", "gpt", "config.yaml")
	}
}

func InitConfig(fileName string) error {
	if fileName == "" {
		fileName = DefaultConfigPath()
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

func LoadConfig(fileName string) (*AppConf, error) {
	if fileName == "" {
		fileName = DefaultConfigPath()
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
