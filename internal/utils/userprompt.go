package utils

import (
	"os"
	"path"
	"regexp"
	"strings"
)

func UserPrompt(args []string) string {

	var buf strings.Builder

	confPath := DefaultConfigPath()

	tryReadFile := func(filePath string) {
		fi, err := os.Stat(filePath)
		if err == nil && !fi.IsDir() {
			content, err := os.ReadFile(filePath)
			if err == nil {
				buf.WriteString(string(content))
				buf.WriteString(" ")
			}
		}
	}

	for _, arg := range args {
		tryReadFile(arg)
		tryReadFile(path.Join(confPath, arg))
		buf.WriteString(arg)
		buf.WriteString(" ")
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
