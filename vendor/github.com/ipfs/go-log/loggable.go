package log

// Loggable describes objects that can be marshalled into Metadata for logging
type Loggable interface {
	Loggable() map[string]interface{}
}

// LoggableMap is just a generic map keyed by string. It
// implements the Loggable interface.
type LoggableMap map[string]interface{}

// Loggable implements the Loggable interface for LoggableMap
func (l LoggableMap) Loggable() map[string]interface{} {
	return l
}

// LoggableF converts a func into a Loggable
type LoggableF func() map[string]interface{}

// Loggable implements the Loggable interface by running
// the LoggableF function.
func (l LoggableF) Loggable() map[string]interface{} {
	return l()
}

// Deferred returns a LoggableF where the execution of the
// provided function is deferred.
func Deferred(key string, f func() string) Loggable {
	function := func() map[string]interface{} {
		return map[string]interface{}{
			key: f(),
		}
	}
	return LoggableF(function)
}

// Pair returns a Loggable where key is paired to Loggable.
func Pair(key string, l Loggable) Loggable {
	return LoggableMap{
		key: l,
	}
}
