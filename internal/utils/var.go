package utils

import "regexp"

var variableRegex = regexp.MustCompile(`(\$\{[^\}]+\})`)

func ExpandVariables(s string, variables ...map[string]string) string {

	return variableRegex.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-1]
		for _, vars := range variables {
			if val, ok := vars[name]; ok {
				return val
			}
		}
		return match
	})
}
