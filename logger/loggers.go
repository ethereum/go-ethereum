/*
Package logger implements a multi-output leveled logger.

Other packages use tagged logger to send log messages to shared
(process-wide) logging engine. The shared logging engine dispatches to
multiple log systems. The log level can be set separately per log
system.

Logging is asynchronous and does not block the caller. Message
formatting is performed by the caller goroutine to avoid incorrect
logging of mutable state.
*/
package logger

import (
	"encoding/json"
	"fmt"
	"os"
)

type LogLevel uint32

const (
	// Standard log levels
	Silence LogLevel = iota
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	DebugDetailLevel
)

// A Logger prints messages prefixed by a given tag. It provides named
// Printf and Println style methods for all loglevels. Each ethereum
// component should have its own logger with a unique prefix.
type Logger struct {
	tag string
}

func NewLogger(tag string) *Logger {
	return &Logger{"[" + tag + "] "}
}

func (logger *Logger) Sendln(level LogLevel, v ...interface{}) {
	logMessageC <- stdMsg{level, logger.tag + fmt.Sprintln(v...)}
}

func (logger *Logger) Sendf(level LogLevel, format string, v ...interface{}) {
	logMessageC <- stdMsg{level, logger.tag + fmt.Sprintf(format, v...)}
}

// Errorln writes a message with ErrorLevel.
func (logger *Logger) Errorln(v ...interface{}) {
	logger.Sendln(ErrorLevel, v...)
}

// Warnln writes a message with WarnLevel.
func (logger *Logger) Warnln(v ...interface{}) {
	logger.Sendln(WarnLevel, v...)
}

// Infoln writes a message with InfoLevel.
func (logger *Logger) Infoln(v ...interface{}) {
	logger.Sendln(InfoLevel, v...)
}

// Debugln writes a message with DebugLevel.
func (logger *Logger) Debugln(v ...interface{}) {
	logger.Sendln(DebugLevel, v...)
}

// DebugDetailln writes a message with DebugDetailLevel.
func (logger *Logger) DebugDetailln(v ...interface{}) {
	logger.Sendln(DebugDetailLevel, v...)
}

// Errorf writes a message with ErrorLevel.
func (logger *Logger) Errorf(format string, v ...interface{}) {
	logger.Sendf(ErrorLevel, format, v...)
}

// Warnf writes a message with WarnLevel.
func (logger *Logger) Warnf(format string, v ...interface{}) {
	logger.Sendf(WarnLevel, format, v...)
}

// Infof writes a message with InfoLevel.
func (logger *Logger) Infof(format string, v ...interface{}) {
	logger.Sendf(InfoLevel, format, v...)
}

// Debugf writes a message with DebugLevel.
func (logger *Logger) Debugf(format string, v ...interface{}) {
	logger.Sendf(DebugLevel, format, v...)
}

// DebugDetailf writes a message with DebugDetailLevel.
func (logger *Logger) DebugDetailf(format string, v ...interface{}) {
	logger.Sendf(DebugDetailLevel, format, v...)
}

// Fatalln writes a message with ErrorLevel and exits the program.
func (logger *Logger) Fatalln(v ...interface{}) {
	logger.Sendln(ErrorLevel, v...)
	Flush()
	os.Exit(0)
}

// Fatalf writes a message with ErrorLevel and exits the program.
func (logger *Logger) Fatalf(format string, v ...interface{}) {
	logger.Sendf(ErrorLevel, format, v...)
	Flush()
	os.Exit(0)
}

type JsonLogger struct {
	Coinbase string
}

func NewJsonLogger() *JsonLogger {
	return &JsonLogger{}
}

func (logger *JsonLogger) LogJson(v JsonLog) {
	msgname := v.EventName()
	obj := map[string]interface{}{
		msgname: v,
	}

	jsontxt, _ := json.Marshal(obj)
	logMessageC <- (jsonMsg(jsontxt))

}
