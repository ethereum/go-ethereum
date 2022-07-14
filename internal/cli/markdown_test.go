package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodeBlock(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	lines := []string{
		"abc",
		"bcd",
	}

	expected := "```\n" + "abc\n" + "bcd\n" + "```"
	assert.Equal(expected, CodeBlock(lines))
}
