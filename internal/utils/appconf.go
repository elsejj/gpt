package utils

import (
	"strings"

	"github.com/elsejj/gpt/internal/mcps"
	"github.com/spf13/viper"
)

// LLM defines the configuration for a large language model.
type LLM struct {
	Gateway      string `yaml:"gateway,omitempty" json:"gateway,omitempty"`
	ApiKey       string `yaml:"apiKey,omitempty" json:"apiKey,omitempty"`
	Provider     string `yaml:"provider" json:"provider"`
	Model        string `yaml:"model" json:"model"`
	ReasonEffort string `yaml:"reasonEffort,omitempty" json:"reasonEffort,omitempty"`
}

// Prompt defines the structure of a user prompt.
type Prompt struct {
	System        string
	Images        []string
	User          string
	WithUsage     bool
	JsonMode      bool
	OverrideModel string
	OnlyCodeBlock bool
	Temperature   float64
	MCPServers    *mcps.MCPs
}

// AppConf defines the application's configuration.
type AppConf struct {
	LLM    LLM            `yaml:"llm" json:"llm"`
	LLMs   map[string]LLM `yaml:"llms,omitempty" json:"llms,omitempty"`
	Prompt *Prompt
}

func parseReasonEffort(effort string) string {
	switch strings.ToLower(effort) {
	case "1", "minimal":
		return "minimal"
	case "2", "low":
		return "low"
	case "3", "medium":
		return "medium"
	case "4", "high":
		return "high"
	case "0", "none":
		return "none"
	default:
		return ""
	}
}

// PickupModel overrides the default model with the one provided by the user.
func (c *AppConf) PickupModel() {
	reasonEffort := parseReasonEffort(viper.GetString("reason"))
	if c.Prompt.OverrideModel != "" {
		model, provider, _ := strings.Cut(c.Prompt.OverrideModel, ":")

		if len(model) > 0 && len(provider) > 0 {
			// override model and provider are all provided, user want to change model and provider
			c.LLM.Model = model
			c.LLM.Provider = provider
			if len(reasonEffort) > 0 {
				c.LLM.ReasonEffort = reasonEffort
			}
			return
		}
		if len(model) > 0 {
			// only model is provided, user want to change model, model can be in llms or just the model name
			if llm, ok := c.LLMs[model]; ok {
				// model is in llms
				if len(llm.Provider) > 0 {
					c.LLM.Provider = llm.Provider
				}
				if len(llm.Model) > 0 {
					c.LLM.Model = llm.Model
				}
				if len(llm.ApiKey) > 0 {
					c.LLM.ApiKey = llm.ApiKey
				}
				if len(llm.Gateway) > 0 {
					c.LLM.Gateway = llm.Gateway
				}
				if len(reasonEffort) > 0 {
					c.LLM.ReasonEffort = reasonEffort
				}
				return
			} else {
				// model is not in llms, user just want to change model
				c.LLM.Model = model
				c.LLM.Provider = ""
				if len(reasonEffort) > 0 {
					c.LLM.ReasonEffort = reasonEffort
				}
			}
		}
		if len(provider) > 0 {
			c.LLM.Provider = provider
		}
	}
}
