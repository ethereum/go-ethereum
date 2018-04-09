package log

import (
	"io"

	logging "github.com/whyrusleeping/go-logging"
)

// WriterGroup is the global writer group for logs to output to
var WriterGroup = NewMirrorWriter()

// Option is a generic function
type Option func()

// Configure applies the provided options sequentially from left to right
func Configure(options ...Option) {
	for _, f := range options {
		f()
	}
}

// LdJSONFormatter Option formats the event log as line-delimited JSON
var LdJSONFormatter = func() {
	logging.SetFormatter(&PoliteJSONFormatter{})
}

// TextFormatter Option formats the event log as human-readable plain-text
var TextFormatter = func() {
	logging.SetFormatter(logging.DefaultFormatter)
}

// Output returns an option which sets the the given writer as the new
// logging backend
func Output(w io.Writer) Option {
	return func() {
		backend := logging.NewLogBackend(w, "", 0)
		logging.SetBackend(backend)
		// TODO return previous Output option
	}
}

// LevelDebug Option sets the log level to debug
var LevelDebug = func() {
	logging.SetLevel(logging.DEBUG, "")
}

// LevelError Option sets the log level to error
var LevelError = func() {
	logging.SetLevel(logging.ERROR, "")
}

// LevelInfo Option sets the log level to info
var LevelInfo = func() {
	logging.SetLevel(logging.INFO, "")
}
