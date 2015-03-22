package test

import (
	"log"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/logger"
)

// logging in tests

var once sync.Once

/* usage:
func TestFunc(t *testing.T) {
    test.LogInit()
    // test
}
*/
func LogInit() {
	once.Do(func() {
		logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.LogLevel(logger.DebugDetailLevel))
	})
}

type testLogger struct{ t *testing.T }

/* usage:
func TestFunc(t *testing.T) {
    defer test.Testlog.Detach()
    // test
}
*/
func Testlog(t *testing.T) testLogger {
	logger.Reset()
	l := testLogger{t}
	logger.AddLogSystem(l)
	return l
}

func (l testLogger) LogPrint(msg logger.LogMsg) {
	l.t.Log(msg.String())
}

func (testLogger) Detach() {
	logger.Flush()
	logger.Reset()
}

type benchLogger struct{ b *testing.B }

/* usage:
func BenchmarkFunc(b *testing.B) {
    defer test.Benchlog.Detach()
    // test
}
*/
func Benchlog(b *testing.B) benchLogger {
	logger.Reset()
	l := benchLogger{b}
	logger.AddLogSystem(l)
	return l
}

func (l benchLogger) LogPrint(msg logger.LogMsg) {
	l.b.Log(msg.String())
}

func (benchLogger) Detach() {
	logger.Flush()
	logger.Reset()
}
