package utils

import (
	"os"
	"regexp"
	"strings"
)

func UserPrompt(args []string) string {

	var buf strings.Builder

	// if arg is a file, read file content
	for _, arg := range args {
		fi, err := os.Stat(arg)
		if err == nil && !fi.IsDir() {
			content, err := os.ReadFile(arg)
			if err == nil {
				buf.Write(content)
				buf.WriteString(" ")
				continue
			}
		}
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
