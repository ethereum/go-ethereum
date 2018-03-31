// +build windows plan9

package poll

import (
	"errors"
)

func WaitWrite(fd int) error {
	return errors.New("platform not supported")
}
