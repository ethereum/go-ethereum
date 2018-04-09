package ipnet

// PNetError is error type for ease of detecting PNet errors
type PNetError interface {
	IsPNetError() bool
}

// NewError creates new PNetError
func NewError(err string) error {
	return pnetErr("privnet: " + err)
}

// IsPNetError checks if given error is PNet Error
func IsPNetError(err error) bool {
	v, ok := err.(PNetError)
	return ok && v.IsPNetError()
}

type pnetErr string

var _ PNetError = (PNetError)(pnetErr(""))

func (p pnetErr) Error() string {
	return string(p)
}

func (pnetErr) IsPNetError() bool {
	return true
}
