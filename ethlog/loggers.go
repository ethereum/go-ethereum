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
	LogPrint(LogLevel, string)
}

type message struct {
	level LogLevel
	msg   string
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
	logMessageC = make(chan message)
	addSystemC  = make(chan LogSystem)
	flushC      = make(chan chan struct{})
	resetC      = make(chan chan struct{})
)

func init() {
	go dispatchLoop()
}

// each system can buffer this many messages before
// blocking incoming log messages.
const sysBufferSize = 500

func dispatchLoop() {
	var (
		systems  []LogSystem
		systemIn []chan message
		systemWG sync.WaitGroup
	)
	bootSystem := func(sys LogSystem) {
		in := make(chan message, sysBufferSize)
		systemIn = append(systemIn, in)
		systemWG.Add(1)
		go sysLoop(sys, in, &systemWG)
	}

	for {
		select {
		case msg := <-logMessageC:
			for _, c := range systemIn {
				c <- msg
			}

		case sys := <-addSystemC:
			systems = append(systems, sys)
			bootSystem(sys)

		case waiter := <-resetC:
			// reset means terminate all systems
			for _, c := range systemIn {
				close(c)
			}
			systems = nil
			systemIn = nil
			systemWG.Wait()
			close(waiter)

		case waiter := <-flushC:
			// flush means reboot all systems
			for _, c := range systemIn {
				close(c)
			}
			systemIn = nil
			systemWG.Wait()
			for _, sys := range systems {
				bootSystem(sys)
			}
			close(waiter)
		}
	}
}

func sysLoop(sys LogSystem, in <-chan message, wg *sync.WaitGroup) {
	for msg := range in {
		if sys.GetLogLevel() >= msg.level {
			sys.LogPrint(msg.level, msg.msg)
		}
	}
	wg.Done()
}

// Reset removes all active log systems.
// It blocks until all current messages have been delivered.
func Reset() {
	waiter := make(chan struct{})
	resetC <- waiter
	<-waiter
}

// Flush waits until all current log messages have been dispatched to
// the active log systems.
func Flush() {
	waiter := make(chan struct{})
	flushC <- waiter
	<-waiter
}

// AddLogSystem starts printing messages to the given LogSystem.
func AddLogSystem(sys LogSystem) {
	addSystemC <- sys
}

// A Logger prints messages prefixed by a given tag. It provides named
// Printf and Println style methods for all loglevels. Each ethereum
// component should have its own logger with a unique prefix.
type Logger struct {
	tag string
}

func NewLogger(tag string) *Logger {
	return &Logger{"[" + tag + "] "}
}

func (logger *Logger) sendln(level LogLevel, v ...interface{}) {
	logMessageC <- message{level, logger.tag + fmt.Sprintln(v...)}
}

func (logger *Logger) sendf(level LogLevel, format string, v ...interface{}) {
	logMessageC <- message{level, logger.tag + fmt.Sprintf(format, v...)}
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

func (t *stdLogSystem) LogPrint(level LogLevel, msg string) {
	t.logger.Print(msg)
}

func (t *stdLogSystem) SetLogLevel(i LogLevel) {
	atomic.StoreUint32(&t.level, uint32(i))
}

func (t *stdLogSystem) GetLogLevel() LogLevel {
	return LogLevel(atomic.LoadUint32(&t.level))
}
