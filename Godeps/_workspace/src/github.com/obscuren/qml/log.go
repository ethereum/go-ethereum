package qml

// #include "capi.h"
//
import "C"

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

// SetLogger sets the target for messages logged by the qml package,
// including console.log and related calls from within qml code.
//
// The logger value must implement either the StdLogger interface,
// which is satisfied by the standard *log.Logger type, or the QmlLogger
// interface, which offers more control over the logged message.
//
// If no logger is provided, the qml package will send messages to the
// default log package logger. This behavior may also be restored by
// providing a nil logger to this function.
func SetLogger(logger interface{}) {
	if logger == nil {
		logHandler = defaultLogger{}
		return
	}
	if qmll, ok := logger.(QmlLogger); ok {
		logHandler = qmll
		return
	}
	if stdl, ok := logger.(StdLogger); ok {
		logHandler = wrappedStdLogger{stdl}
		return
	}
	panic("unsupported logger interface")
}

// The QmlLogger interface may be implemented to better control how
// log messages from the qml package are handled. Values that
// implement either StdLogger or QmlLogger may be provided to the
// SetLogger function.
type QmlLogger interface {
	// QmlOutput is called whenever a new message is available for logging.
	// The message value must not be used after the method returns.
	QmlOutput(message LogMessage) error
}

// The StdLogger interface is implemented by standard *log.Logger values.
// Values that implement either StdLogger or QmlLogger may be provided
// to the SetLogger function.
type StdLogger interface {
	// Output is called whenever a new message is available for logging.
	// See the standard log.Logger type for more details.
	Output(calldepth int, s string) error
}

// NOTE: LogMessage is an interface to avoid allocating and copying
// several strings for each logged message.

// LogMessage is implemented by values provided to QmlLogger.QmlOutput.
type LogMessage interface {
	Severity() LogSeverity
	Text() string
	File() string
	Line() int

	String() string // returns "file:line: text"

	privateMarker()
}

type LogSeverity int

const (
	LogDebug LogSeverity = iota
	LogWarning
	LogCritical
	LogFatal
)

var logHandler QmlLogger = defaultLogger{}

type defaultLogger struct{}

func (defaultLogger) QmlOutput(msg LogMessage) error {
	log.Println(msg.String())
	return nil
}

func init() {
	// Install the C++ log handler that diverts calls to the hook below.
	C.installLogHandler()
}

//export hookLogHandler
func hookLogHandler(cmsg *C.LogMessage) {
	// Workarund for QTBUG-35943
	text := unsafeString(cmsg.text, cmsg.textLen)
	if strings.HasPrefix(text, `"Qt Warning: Compose file:`) {
		return
	}
	msg := logMessage{c: cmsg}
	logHandler.QmlOutput(&msg)
	msg.invalid = true
}

type wrappedStdLogger struct {
	StdLogger
}

func (l wrappedStdLogger) QmlOutput(msg LogMessage) error {
	return l.Output(0, msg.String())
}

type logMessage struct {
	c *C.LogMessage

	// invalid flags that cmsg points to unreliable memory,
	// since the log hook has already returned.
	invalid bool
}

func (m *logMessage) assertValid() {
	if m.invalid {
		panic("attempted to use log message outside of log hook")
	}
}

func (m *logMessage) Severity() LogSeverity {
	return LogSeverity(m.c.severity)
}

func (m *logMessage) Line() int {
	m.assertValid()
	return int(m.c.line)
}

func (m *logMessage) String() string {
	m.assertValid()
	file := unsafeString(m.c.file, m.c.fileLen)
	text := unsafeString(m.c.text, m.c.textLen)
	return fmt.Sprintf("%s:%d: %s", filepath.Base(file), m.c.line, text)
}

func (m *logMessage) File() string {
	m.assertValid()
	return C.GoStringN(m.c.file, m.c.fileLen)
}

func (m *logMessage) Text() string {
	m.assertValid()
	return C.GoStringN(m.c.text, m.c.line)
}

func (*logMessage) privateMarker() {}
