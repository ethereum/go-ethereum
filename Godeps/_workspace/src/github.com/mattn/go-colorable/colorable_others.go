// +build !windows

package colorable

import (
	"io"
	"os"
)

func NewColorableStdout() io.Writer {
	return os.Stdout
}

func NewColorableStderr() io.Writer {
	return os.Stderr
}
