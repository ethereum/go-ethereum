package test

import (
	"log"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/logger"
)

var once sync.Once

/* usage:
func TestFunc(t *testing.T) {
    test.LogInit()
    // test
}
*/
func LogInit() {
	once.Do(func() {
		var logsys = logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.LogLevel(logger.DebugDetailLevel))
		logger.AddLogSystem(logsys)
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

func (testLogger) GetLogLevel() logger.LogLevel { return logger.DebugLevel }
func (testLogger) SetLogLevel(logger.LogLevel)  {}

func (l testLogger) LogPrint(level logger.LogLevel, msg string) {
	l.t.Logf("%s", msg)
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

func (benchLogger) GetLogLevel() logger.LogLevel { return logger.Silence }

func (benchLogger) SetLogLevel(logger.LogLevel) {}
func (l benchLogger) LogPrint(level logger.LogLevel, msg string) {
	l.b.Logf("%s", msg)
}
func (benchLogger) Detach() {
	logger.Flush()
	logger.Reset()
}
