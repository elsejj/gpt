package tools

import (
	"encoding/json"
	"fmt"
	"os"
	filepath "path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/elsejj/gpt/internal/utils"
	"github.com/goccy/go-yaml"
)

// Tool represents the configuration for a language model tool, when specified, will override global settings.
type Tool struct {
	Model        string   `yaml:"model,omitempty" json:"model,omitempty" toml:"model,omitempty"`
	Key          string   `yaml:"key,omitempty" json:"key,omitempty" toml:"key,omitempty"`
	URL          string   `yaml:"url,omitempty" json:"url,omitempty" toml:"url,omitempty"`
	ReasonEffort string   `yaml:"reason,omitempty" json:"reason,omitempty" toml:"reason,omitempty"`
	Temperature  *float64 `yaml:"temperature,omitempty" json:"temperature,omitempty" toml:"temperature,omitempty"`
	SystemPrompt string   `yaml:"system,omitempty" json:"system,omitempty" toml:"system,omitempty"`
	UserTemplate string   `yaml:"user,omitempty" json:"user,omitempty" toml:"user,omitempty"`
	MCPs         []string `yaml:"mcps,omitempty" json:"mcps,omitempty" toml:"mcps,omitempty"`
}

var parsers = map[string]func([]byte, *Tool) error{
	".yaml": parseYAML,
	".yml":  parseYAML,
	".json": parseJSON,
	".toml": parseTOML,
}

func Load(name string) (Tool, error) {
	var tool Tool
	path, err := findFile(name)
	if err != nil {
		return tool, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	parser, ok := parsers[ext]
	if !ok {
		return tool, fmt.Errorf("%s is not a supported format", name)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return tool, err
	}
	err = parser(data, &tool)
	return tool, err
}

func (tool *Tool) UserPrompt(user string) string {
	if tool.UserTemplate == "" {
		return user
	}
	if strings.Contains(tool.UserTemplate, "{{user}}") {
		return strings.ReplaceAll(tool.UserTemplate, "{{user}}", user)
	}
	return tool.UserTemplate + "\n" + user
}

func findFile(name string) (string, error) {
	tryFiles := []string{}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		for k := range parsers {
			tryFiles = append(tryFiles, name+k)
			tryFiles = append(tryFiles, utils.ConfigPath("tools", name+k))
		}
	} else {
		tryFiles = append(tryFiles, name)
		tryFiles = append(tryFiles, utils.ConfigPath("tools", name))
	}

	for _, f := range tryFiles {
		if _, err := os.Stat(f); err == nil {
			return f, nil
		}
	}
	return "", fmt.Errorf("tool config file %s not found", name)
}

func parseJSON(data []byte, tool *Tool) error {
	return json.Unmarshal(data, tool)
}

func parseYAML(data []byte, tool *Tool) error {
	return yaml.Unmarshal(data, tool)
}

func parseTOML(data []byte, tool *Tool) error {
	return toml.Unmarshal(data, tool)
}
