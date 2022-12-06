package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodeBlock(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	lines := []string{
		"abc",
		"bcd",
	}

	expected := "```\n" + "abc\n" + "bcd\n" + "```"
	assert.Equal(expected, CodeBlock(lines))
}
