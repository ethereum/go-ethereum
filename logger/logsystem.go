package logger

import (
	"io"
	"log"
	"sync/atomic"
)

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

func (t *stdLogSystem) LogPrint(level LogLevel, msg string) {
	t.logger.Print(msg)
}

func (t *stdLogSystem) SetLogLevel(i LogLevel) {
	atomic.StoreUint32(&t.level, uint32(i))
}

func (t *stdLogSystem) GetLogLevel() LogLevel {
	return LogLevel(atomic.LoadUint32(&t.level))
}

// NewRawLogSystem creates a LogSystem that prints to the given writer without
// adding extra information. Suitable for preformatted output
func NewRawLogSystem(writer io.Writer, flags int, level LogLevel) LogSystem {
	logger := log.New(writer, "", 0)
	return &rawLogSystem{logger, uint32(level)}
}

type rawLogSystem struct {
	logger *log.Logger
	level  uint32
}

func (t *rawLogSystem) LogPrint(level LogLevel, msg string) {
	t.logger.Print(msg)
}

func (t *rawLogSystem) SetLogLevel(i LogLevel) {
	atomic.StoreUint32(&t.level, uint32(i))
}

func (t *rawLogSystem) GetLogLevel() LogLevel {
	return LogLevel(atomic.LoadUint32(&t.level))
}
