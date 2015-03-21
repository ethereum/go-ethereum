package logger

import (
	"io"
	"log"
	"sync/atomic"
)

// LogSystem is implemented by log output devices.
// All methods can be called concurrently from multiple goroutines.
type LogSystem interface {
	LogPrint(LogMsg)
}

// NewStdLogSystem creates a LogSystem that prints to the given writer.
// The flag values are defined package log.
func NewStdLogSystem(writer io.Writer, flags int, level LogLevel) LogSystem {
	logger := log.New(writer, "", flags)
	return &stdLogSystem{logger, uint32(level)}
}

type stdLogSystem struct {
	logger *log.Logger
	level  uint32
}

func (t *stdLogSystem) LogPrint(msg LogMsg) {
	stdmsg, ok := msg.(stdMsg)
	if ok {
		if t.GetLogLevel() >= stdmsg.Level() {
			t.logger.Print(stdmsg.String())
		}
	}
}

func (t *stdLogSystem) SetLogLevel(i LogLevel) {
	atomic.StoreUint32(&t.level, uint32(i))
}

func (t *stdLogSystem) GetLogLevel() LogLevel {
	return LogLevel(atomic.LoadUint32(&t.level))
}

// NewJSONLogSystem creates a LogSystem that prints to the given writer without
// adding extra information irrespective of loglevel only if message is JSON type
func NewJsonLogSystem(writer io.Writer) LogSystem {
	logger := log.New(writer, "", 0)
	return &jsonLogSystem{logger}
}

type jsonLogSystem struct {
	logger *log.Logger
}

func (t *jsonLogSystem) LogPrint(msg LogMsg) {
	jsonmsg, ok := msg.(jsonMsg)
	if ok {
		t.logger.Print(jsonmsg.String())
	}
}
