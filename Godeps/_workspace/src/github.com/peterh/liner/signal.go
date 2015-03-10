// +build go1.1,!windows

package liner

import (
	"os"
	"os/signal"
)

func stopSignal(c chan<- os.Signal) {
	signal.Stop(c)
}
