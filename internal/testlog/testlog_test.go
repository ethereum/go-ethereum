package testlog

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

func TestLogging(t *testing.T) {
	l := Logger(t, log.LevelInfo)
	subLogger := l.New("foobar", 123)

	l.Info("Visible")
	subLogger.Info("Hide and seek") // this log is erroneously hidden in master, but fixed with this PR
	l.Info("Also visible")
}
