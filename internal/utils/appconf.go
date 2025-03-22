package utils

import (
	"strings"

	"github.com/elsejj/gpt/internal/mcps"
)

type LLM struct {
	Gateway  string `yaml:"gateway,omitempty" json:"gateway,omitempty"`
	ApiKey   string `yaml:"apiKey,omitempty" json:"apiKey,omitempty"`
	Provider string `yaml:"provider" json:"provider"`
	Model    string `yaml:"model" json:"model"`
}

type Prompt struct {
	System        string
	Images        []string
	User          string
	WithUsage     bool
	JsonMode      bool
	OverrideModel string
	OnlyCodeBlock bool
	MCPServers    *mcps.MCPs
}

type AppConf struct {
	LLM    LLM            `yaml:"llm" json:"llm"`
	LLMs   map[string]LLM `yaml:"llms,omitempty" json:"llms,omitempty"`
	Prompt *Prompt
}

func (c *AppConf) PickupModel() {
	if c.Prompt.OverrideModel != "" {
		model, provider, _ := strings.Cut(c.Prompt.OverrideModel, ":")

		if len(model) > 0 && len(provider) > 0 {
			// override model and provider are all provided, user want to change model and provider
			c.LLM.Model = model
			c.LLM.Provider = provider
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
				return
			} else {
				// model is not in llms, user just want to change model
				c.LLM.Model = model
				c.LLM.Provider = ""
			}
		}
		if len(provider) > 0 {
			c.LLM.Provider = provider
		}
	}
}
