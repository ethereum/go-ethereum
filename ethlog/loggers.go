package ethlog

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

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

// Reset removes all registered log systems.
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

type Logger struct {
	tag string
}

func NewLogger(tag string) *Logger {
	return &Logger{tag}
}

func AddLogSystem(logSystem LogSystem) {
	mutex.Lock()
	logSystems = append(logSystems, logSystem)
	mutex.Unlock()
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

func (logger *Logger) Errorln(v ...interface{}) {
	logger.sendln(ErrorLevel, v...)
}

func (logger *Logger) Warnln(v ...interface{}) {
	logger.sendln(WarnLevel, v...)
}

func (logger *Logger) Infoln(v ...interface{}) {
	logger.sendln(InfoLevel, v...)
}

func (logger *Logger) Debugln(v ...interface{}) {
	logger.sendln(DebugLevel, v...)
}

func (logger *Logger) DebugDetailln(v ...interface{}) {
	logger.sendln(DebugDetailLevel, v...)
}

func (logger *Logger) Errorf(format string, v ...interface{}) {
	logger.sendf(ErrorLevel, format, v...)
}

func (logger *Logger) Warnf(format string, v ...interface{}) {
	logger.sendf(WarnLevel, format, v...)
}

func (logger *Logger) Infof(format string, v ...interface{}) {
	logger.sendf(InfoLevel, format, v...)
}

func (logger *Logger) Debugf(format string, v ...interface{}) {
	logger.sendf(DebugLevel, format, v...)
}

func (logger *Logger) DebugDetailf(format string, v ...interface{}) {
	logger.sendf(DebugDetailLevel, format, v...)
}

func (logger *Logger) Fatalln(v ...interface{}) {
	logger.sendln(ErrorLevel, v...)
	Flush()
	os.Exit(0)
}

func (logger *Logger) Fatalf(format string, v ...interface{}) {
	logger.sendf(ErrorLevel, format, v...)
	Flush()
	os.Exit(0)
}

type StdLogSystem struct {
	logger *log.Logger
	level  LogLevel
}

func (t *StdLogSystem) Println(v ...interface{}) {
	t.logger.Println(v...)
}

func (t *StdLogSystem) Printf(format string, v ...interface{}) {
	t.logger.Printf(format, v...)
}

func (t *StdLogSystem) SetLogLevel(i LogLevel) {
	t.level = i
}

func (t *StdLogSystem) GetLogLevel() LogLevel {
	return t.level
}

func NewStdLogSystem(writer io.Writer, flags int, level LogLevel) *StdLogSystem {
	logger := log.New(writer, "", flags)
	return &StdLogSystem{logger, level}
}
