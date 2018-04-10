package log

import (
	"errors"
	"fmt"
	"os"

	colorable "github.com/mattn/go-colorable"
	logging "github.com/whyrusleeping/go-logging"
)

func init() {
	SetupLogging()
}

var ansiGray = "\033[0;37m"
var ansiBlue = "\033[0;34m"

// LogFormats defines formats for logging (i.e. "color")
var LogFormats = map[string]string{
	"nocolor": "%{time:2006-01-02 15:04:05.000000} %{level} %{module} %{shortfile}: %{message}",
	"color": ansiGray + "%{time:15:04:05.000} %{color}%{level:5.5s} " + ansiBlue +
		"%{module:10.10s}: %{color:reset}%{message} " + ansiGray + "%{shortfile}%{color:reset}",
}

var defaultLogFormat = "color"

// Logging environment variables
const (
	envLogging    = "IPFS_LOGGING"
	envLoggingFmt = "IPFS_LOGGING_FMT"
)

// ErrNoSuchLogger is returned when the util pkg is asked for a non existant logger
var ErrNoSuchLogger = errors.New("Error: No such logger")

// loggers is the set of loggers in the system
var loggers = map[string]*logging.Logger{}

// SetupLogging will initialize the logger backend and set the flags.
func SetupLogging() {

	lfmt := LogFormats[os.Getenv(envLoggingFmt)]
	if lfmt == "" {
		lfmt = LogFormats[defaultLogFormat]
	}

	backend := logging.NewLogBackend(colorable.NewColorableStderr(), "", 0)
	logging.SetBackend(backend)
	logging.SetFormatter(logging.MustStringFormatter(lfmt))

	lvl := logging.ERROR

	if logenv := os.Getenv(envLogging); logenv != "" {
		var err error
		lvl, err = logging.LogLevel(logenv)
		if err != nil {
			fmt.Println("error setting log levels", err)
		}
	}

	SetAllLoggers(lvl)
}

// SetDebugLogging calls SetAllLoggers with logging.DEBUG
func SetDebugLogging() {
	SetAllLoggers(logging.DEBUG)
}

// SetAllLoggers changes the logging.Level of all loggers to lvl
func SetAllLoggers(lvl logging.Level) {
	logging.SetLevel(lvl, "")
	for n := range loggers {
		logging.SetLevel(lvl, n)
	}
}

// SetLogLevel changes the log level of a specific subsystem
// name=="*" changes all subsystems
func SetLogLevel(name, level string) error {
	lvl, err := logging.LogLevel(level)
	if err != nil {
		return err
	}

	// wildcard, change all
	if name == "*" {
		SetAllLoggers(lvl)
		return nil
	}

	// Check if we have a logger by that name
	if _, ok := loggers[name]; !ok {
		return ErrNoSuchLogger
	}

	logging.SetLevel(lvl, name)

	return nil
}

// GetSubsystems returns a slice containing the
// names of the current loggers
func GetSubsystems() []string {
	subs := make([]string, 0, len(loggers))

	for k := range loggers {
		subs = append(subs, k)
	}
	return subs
}

func getLogger(name string) *logging.Logger {
	log := logging.MustGetLogger(name)
	log.ExtraCalldepth = 1
	loggers[name] = log
	return log
}
