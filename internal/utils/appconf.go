package utils

type LLM struct {
	Gateway  string `yaml:"gateway" json:"gateway"`
	ApiKey   string `yaml:"apiKey" json:"apiKey"`
	Provider string `yaml:"provider" json:"provider"`
	Model    string `yaml:"model" json:"model"`
}

type Prompt struct {
	System    string
	Images    []string
	User      string
	WithUsage bool
	JsonMode  bool
}

type AppConf struct {
	LLM    LLM `yaml:"llm" json:"llm"`
	Prompt *Prompt
}
