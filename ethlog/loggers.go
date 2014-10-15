/*
Package ethlog implements a multi-output leveled logger.

Other packages use tagged logger to send log messages to shared
(process-wide) logging engine. The shared logging engine dispatches to
multiple log systems. The log level can be set separately per log
system.

Logging is asynchrounous and does not block the caller. Message
formatting is performed by the caller goroutine to avoid incorrect
logging of mutable state.
*/
package ethlog

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

// LogSystem is implemented by log output devices.
// All methods can be called concurrently from multiple goroutines.
type LogSystem interface {
	GetLogLevel() LogLevel
	SetLogLevel(i LogLevel)
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

type logMessage struct {
	LogLevel LogLevel
	format   bool
	msg      string
}

func newPrintlnLogMessage(level LogLevel, tag string, v ...interface{}) *logMessage {
	return &logMessage{level, false, fmt.Sprintf("[%s] %s", tag, fmt.Sprint(v...))}
}

func newPrintfLogMessage(level LogLevel, tag string, format string, v ...interface{}) *logMessage {
	return &logMessage{level, true, fmt.Sprintf("[%s] %s", tag, fmt.Sprintf(format, v...))}
}

type LogLevel uint8

const (
	// Standard log levels
	Silence LogLevel = iota
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	DebugDetailLevel
)

var (
	mutex      sync.RWMutex // protects logSystems
	logSystems []LogSystem

	logMessages  = make(chan *logMessage)
	drainWaitReq = make(chan chan struct{})
)

func init() {
	go dispatchLoop()
}

func dispatchLoop() {
	var drainWait []chan struct{}
	dispatchDone := make(chan struct{})
	pending := 0
	for {
		select {
		case msg := <-logMessages:
			go dispatch(msg, dispatchDone)
			pending++
		case waiter := <-drainWaitReq:
			if pending == 0 {
				close(waiter)
			} else {
				drainWait = append(drainWait, waiter)
			}
		case <-dispatchDone:
			pending--
			if pending == 0 {
				for _, c := range drainWait {
					close(c)
				}
				drainWait = nil
			}
		}
	}
}

func dispatch(msg *logMessage, done chan<- struct{}) {
	mutex.RLock()
	for _, sys := range logSystems {
		if sys.GetLogLevel() >= msg.LogLevel {
			if msg.format {
				sys.Printf(msg.msg)
			} else {
				sys.Println(msg.msg)
			}
		}
	}
	mutex.RUnlock()
	done <- struct{}{}
}

// send delivers a message to all installed log
// systems. it doesn't wait for the message to be
// written.
func send(msg *logMessage) {
	logMessages <- msg
}

// Reset removes all active log systems.
func Reset() {
	mutex.Lock()
	logSystems = nil
	mutex.Unlock()
}

// Flush waits until all current log messages have been dispatched to
// the active log systems.
func Flush() {
	waiter := make(chan struct{})
	drainWaitReq <- waiter
	<-waiter
}

// AddLogSystem starts printing messages to the given LogSystem.
func AddLogSystem(logSystem LogSystem) {
	mutex.Lock()
	logSystems = append(logSystems, logSystem)
	mutex.Unlock()
}

// A Logger prints messages prefixed by a given tag. It provides named
// Printf and Println style methods for all loglevels. Each ethereum
// component should have its own logger with a unique prefix.
type Logger struct {
	tag string
}

func NewLogger(tag string) *Logger {
	return &Logger{tag}
}

func (logger *Logger) sendln(level LogLevel, v ...interface{}) {
	if logMessages != nil {
		msg := newPrintlnLogMessage(level, logger.tag, v...)
		send(msg)
	}
}

func (logger *Logger) sendf(level LogLevel, format string, v ...interface{}) {
	if logMessages != nil {
		msg := newPrintfLogMessage(level, logger.tag, format, v...)
		send(msg)
	}
}

// Errorln writes a message with ErrorLevel.
func (logger *Logger) Errorln(v ...interface{}) {
	logger.sendln(ErrorLevel, v...)
}

// Warnln writes a message with WarnLevel.
func (logger *Logger) Warnln(v ...interface{}) {
	logger.sendln(WarnLevel, v...)
}

// Infoln writes a message with InfoLevel.
func (logger *Logger) Infoln(v ...interface{}) {
	logger.sendln(InfoLevel, v...)
}

// Debugln writes a message with DebugLevel.
func (logger *Logger) Debugln(v ...interface{}) {
	logger.sendln(DebugLevel, v...)
}

// DebugDetailln writes a message with DebugDetailLevel.
func (logger *Logger) DebugDetailln(v ...interface{}) {
	logger.sendln(DebugDetailLevel, v...)
}

// Errorf writes a message with ErrorLevel.
func (logger *Logger) Errorf(format string, v ...interface{}) {
	logger.sendf(ErrorLevel, format, v...)
}

// Warnf writes a message with WarnLevel.
func (logger *Logger) Warnf(format string, v ...interface{}) {
	logger.sendf(WarnLevel, format, v...)
}

// Infof writes a message with InfoLevel.
func (logger *Logger) Infof(format string, v ...interface{}) {
	logger.sendf(InfoLevel, format, v...)
}

// Debugf writes a message with DebugLevel.
func (logger *Logger) Debugf(format string, v ...interface{}) {
	logger.sendf(DebugLevel, format, v...)
}

// DebugDetailf writes a message with DebugDetailLevel.
func (logger *Logger) DebugDetailf(format string, v ...interface{}) {
	logger.sendf(DebugDetailLevel, format, v...)
}

// Fatalln writes a message with ErrorLevel and exits the program.
func (logger *Logger) Fatalln(v ...interface{}) {
	logger.sendln(ErrorLevel, v...)
	Flush()
	os.Exit(0)
}

// Fatalf writes a message with ErrorLevel and exits the program.
func (logger *Logger) Fatalf(format string, v ...interface{}) {
	logger.sendf(ErrorLevel, format, v...)
	Flush()
	os.Exit(0)
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

func (t *stdLogSystem) Println(v ...interface{}) {
	t.logger.Println(v...)
}

func (t *stdLogSystem) Printf(format string, v ...interface{}) {
	t.logger.Printf(format, v...)
}

func (t *stdLogSystem) SetLogLevel(i LogLevel) {
	atomic.StoreUint32(&t.level, uint32(i))
}

func (t *stdLogSystem) GetLogLevel() LogLevel {
	return LogLevel(atomic.LoadUint32(&t.level))
}
