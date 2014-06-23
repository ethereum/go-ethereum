package ethlog

import (
  "fmt"
  "sync"
  "log"
  "io"
  "os"
)

type LogSystem interface {
  GetLogLevel() LogLevel
  SetLogLevel(i LogLevel)
  Println(v ...interface{})
  Printf(format string, v ...interface{})
}

type logMessage struct {
  LogLevel  LogLevel
  format    bool
  msg       string
}

func newPrintlnLogMessage(level LogLevel, tag string, v...interface{}) *logMessage {
  return &logMessage{level, false, fmt.Sprintf("[%s] %s", tag, fmt.Sprint(v...))}
}

func newPrintfLogMessage(level LogLevel, tag string, format string, v...interface{}) *logMessage {
  return &logMessage{level, true, fmt.Sprintf("[%s] %s", tag, fmt.Sprintf(format, v...))}
}

func (msg *logMessage) send(logger LogSystem) {
  if msg.format {
    logger.Printf(msg.msg)
  } else {
    logger.Println(msg.msg)
  }
}

var logMessages chan(*logMessage)
var logSystems  []LogSystem
var drained = true

type LogLevel uint8

const (
  Silence LogLevel = iota
  ErrorLevel
  WarnLevel
  InfoLevel
  DebugLevel
)

// log messages are dispatched to log writers
func start() {
  for {
    select {
      case msg := <- logMessages:
        for _, logSystem := range logSystems {
          if logSystem.GetLogLevel() >= msg.LogLevel {
            msg.send(logSystem)
          }
        }
      default:
        drained = true
    }
  }
}

// waits until log messages are drained (dispatched to log writers)
func Flush() {
  for !drained {}
}

type Logger struct {
  tag string
}

func NewLogger(tag string) *Logger {
  return &Logger{tag}
}

func AddLogSystem(logSystem LogSystem) {
  var mutex = &sync.Mutex{}
  mutex.Lock()
  defer mutex.Unlock()
  if logSystems == nil {
    logMessages = make(chan *logMessage)
    go start()
  }
  logSystems = append(logSystems, logSystem)
}

func (logger *Logger) sendln(level LogLevel, v...interface{}) {
  if logMessages != nil {
    msg := newPrintlnLogMessage(level, logger.tag, v...)
    drained = false
    logMessages <- msg
  }
}

func (logger *Logger) sendf(level LogLevel, format string, v...interface{}) {
  if logMessages != nil {
    msg := newPrintfLogMessage(level, logger.tag, format, v...)
    drained = false
    logMessages <- msg
  }
}

func (logger *Logger) Errorln(v...interface{}) {
  logger.sendln(ErrorLevel, v...)
}

func (logger *Logger) Warnln(v...interface{}) {
  logger.sendln(WarnLevel, v...)
}

func (logger *Logger) Infoln(v...interface{}) {
  logger.sendln(InfoLevel, v...)
}

func (logger *Logger) Debugln(v...interface{}) {
  logger.sendln(DebugLevel, v...)
}

func (logger *Logger) Errorf(format string, v...interface{}) {
  logger.sendf(ErrorLevel, format, v...)
}

func (logger *Logger) Warnf(format string, v...interface{}) {
  logger.sendf(WarnLevel, format, v...)
}

func (logger *Logger) Infof(format string, v...interface{}) {
  logger.sendf(InfoLevel, format, v...)
}

func (logger *Logger) Debugf(format string, v...interface{}) {
  logger.sendf(DebugLevel, format, v...)
}

func (logger *Logger) Fatalln (v...interface{}) {
  logger.sendln(ErrorLevel, v...)
  Flush()
  os.Exit(0)
}

func (logger *Logger) Fatalf (format string, v...interface{}) {
  logger.sendf(ErrorLevel, format, v...)
  Flush()
  os.Exit(0)
}

type StdLogSystem struct {
  logger *log.Logger
  level LogLevel
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

