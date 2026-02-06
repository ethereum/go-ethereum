package adapters

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
)

// wsAddrPattern is a regex used to read the WebSocket address from the node's
// log
var wsAddrPattern = regexp.MustCompile(`ws://[\d.:]+`)

func matchWSAddr(str string) (string, bool) {
	if !strings.Contains(str, "WebSocket endpoint opened") {
		return "", false
	}

	return wsAddrPattern.FindString(str), true
}

// findWSAddr scans through reader r, looking for the log entry with
// WebSocket address information.
func findWSAddr(r io.Reader, timeout time.Duration) (string, error) {
	ch := make(chan string)

	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			addr, ok := matchWSAddr(s.Text())
			if ok {
				ch <- addr
			}
		}
		close(ch)
	}()

	var wsAddr string
	select {
	case wsAddr = <-ch:
		if wsAddr == "" {
			return "", errors.New("empty result")
		}
	case <-time.After(timeout):
		return "", errors.New("timed out")
	}

	return wsAddr, nil
}
