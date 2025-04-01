package testlog

import (
	"github.com/ethereum/go-ethereum/log"
	"testing"
)

func TestLogging(t *testing.T) {
	l := Logger(t, log.LevelInfo)
	subLogger := l.New("foobar", 123)

	l.Info("Visible")
	subLogger.Info("Hide and seek") // not visible due to sub buffer never being flushed
	l.Info("Also visible")

	t.Log("flushed: ", l.Handler().(*bufHandler).buf)
	t.Log("remaining: ", subLogger.Handler().(*bufHandler).buf)
	// horrible hack to manually bring back the expected log data
	l.Handler().(*bufHandler).buf = subLogger.Handler().(*bufHandler).buf
	l.(*logger).flush()
}
