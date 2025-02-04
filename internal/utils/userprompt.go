package utils

import (
	"os"
	"regexp"
	"strings"
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

	return all
}
