package bigcache

import (
	"log"
	"os"
)

// Logger is invoked when `Config.Verbose=true`
type Logger interface {
	Printf(format string, v ...interface{})
}

// this is a safeguard, breaking on compile time in case
// `log.Logger` does not adhere to our `Logger` interface.
// see https://golang.org/doc/faq#guarantee_satisfies_interface
var _ Logger = &log.Logger{}

// DefaultLogger returns a `Logger` implementation
// backed by stdlib's log
func DefaultLogger() *log.Logger {
	return log.New(os.Stdout, "", log.LstdFlags)
}

func newLogger(custom Logger) Logger {
	if custom != nil {
		return custom
	}

	return DefaultLogger()
}
