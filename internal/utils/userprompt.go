package utils

import (
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/elsejj/gpt/internal/mcps"
)

func SplitContentAndVariables(args []string) ([]string, map[string]string) {
	variables := make(map[string]string)
	content := []string{}

	// find all variables with format `varName=value`
	r := regexp.MustCompile(`([^\}=]+)=([^\}]+)`)
	for _, arg := range args {
		matches := r.FindStringSubmatch(arg)
		if len(matches) == 3 {
			varName := strings.TrimSpace(matches[1])
			varValue := strings.TrimSpace(matches[2])
			variables[varName] = varValue
		} else {
			content = append(content, arg)
		}
	}

	return content, variables
}

// UserPrompt processes the user's prompt.
// It reads files, gets MCP prompts, and replaces variables.
func UserPrompt(variables map[string]string, args ...string) string {

	var buf strings.Builder

	tryReadFile := func(filePath string) bool {
		content, err := os.ReadFile(filePath)
		if err == nil {
			buf.WriteString(string(content))
			buf.WriteString(" ")
			return true
		}
		return false
	}

	tryMcpPrompt := func(promptName string) bool {
		if body, ok := mcps.GetPrompt(promptName); ok {
			buf.WriteString(body)
			buf.WriteString(" ")
			return true
		}
		return false
	}

	for _, arg := range args {
		f1 := tryMcpPrompt(arg)
		if f1 {
			continue
		}
		f2 := tryReadFile(arg)
		f3 := tryReadFile(ConfigPath(arg))
		if !f2 && !f3 {
			buf.WriteString(arg)
			buf.WriteString(" ")
		}
	}

	all := buf.String()

	// replace all word starts with @ to file content
	r := regexp.MustCompile(`@([^\s]+)`)
	all = r.ReplaceAllStringFunc(all, func(s string) string {
		filePath := strings.TrimPrefix(s, "@")
		content, err := os.ReadFile(filePath)
		if err != nil {
			return s
		}
		return string(content)
	})

	// replace all variable with format {{varName}}
	all = ExpandVariables(all, variables, globalVariables)

	return all
}

var globalVariables = map[string]string{
	"OS":    runtime.GOOS,
	"TODAY": time.Now().Format(time.RFC3339),
	"SHELL": shellName(),
}

// shellName returns the name of the current shell.
func shellName() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "bash"
}
