package metrics

// NewHealthcheck constructs a new Healthcheck which will use the given
// function to update its status.
func NewHealthcheck(f func(*Healthcheck)) *Healthcheck {
	return &Healthcheck{nil, f}
}

// Healthcheck is the standard implementation of a Healthcheck and
// stores the status and a function to call to update the status.
type Healthcheck struct {
	err error
	f   func(*Healthcheck)
}

// Check runs the healthcheck function to update the healthcheck's status.
func (h *Healthcheck) Check() {
	h.f(h)
}

// Error returns the healthcheck's status, which will be nil if it is healthy.
func (h *Healthcheck) Error() error {
	return h.err
}

// Healthy marks the healthcheck as healthy.
func (h *Healthcheck) Healthy() {
	h.err = nil
}

// Unhealthy marks the healthcheck as unhealthy.  The error is stored and
// may be retrieved by the Error method.
func (h *Healthcheck) Unhealthy(err error) {
	h.err = err
}
