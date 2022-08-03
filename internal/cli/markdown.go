package cli

import (
	"strings"
)

type MarkDown interface {
	MarkDown() string
}

// Create a Markdown code block from a slice of string, where each string is a line of code
func CodeBlock(lines []string) string {
	return "```\n" + strings.Join(lines, "\n") + "\n```"
}
