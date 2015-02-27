package ui

// ReturnInterface is returned by the Intercom interface when a method is called
type ReturnInterface interface {
	Get(i int) (interface{}, error)
	Size() int
}

// Frontend is the basic interface for calling arbitrary methods on something that
// implements a front end (GUI, CLI, etc)
type Frontend interface {
	// Checks whether a specific method is implemented
	Supports(method string) bool
	// Call calls the given method on interface it implements. This will return
	// an error with errNotImplemented if the method hasn't been implemented
	// and will return a ReturnInterface if it does.
	Call(method string) (ReturnInterface, error)
}
