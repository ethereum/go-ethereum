// +build !go1.1,!windows

package liner

import (
	"os"
)

func stopSignal(c chan<- os.Signal) {
	// signal.Stop does not exist before Go 1.1
}
