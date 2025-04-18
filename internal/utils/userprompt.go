package utils

import (
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func UserPrompt(args []string) string {

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

	for _, arg := range args {
		f1 := tryReadFile(arg)
		f2 := tryReadFile(ConfigPath(arg))
		if !f1 && !f2 {
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
	rVar := regexp.MustCompile(`{{([^\}]+)}}`)
	all = rVar.ReplaceAllStringFunc(all, func(s string) string {
		varName := strings.TrimPrefix(s, "{{")
		varName = strings.TrimSuffix(varName, "}}")
		varName = strings.TrimSpace(varName)

		varValue, ok := getVariableValue(varName)
		if ok {
			return varValue
		}
		return s
	})

	return all
}

var globalVariables = map[string]string{
	"OS":    runtime.GOOS,
	"TODAY": time.Now().Format(time.RFC3339),
	"SHELL": shellName(),
}

func shellName() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "bash"
}

func getVariableValue(varName string) (string, bool) {
	if val, ok := globalVariables[strings.ToUpper(varName)]; ok {
		return val, true
	}
	return "", false
}
