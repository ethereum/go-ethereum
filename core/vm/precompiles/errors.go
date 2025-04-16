package precompiles

import "errors"

var (
	ErrSenderNotOrigin   = errors.New("sender not origin")
	ErrSenderNotRegistry = errors.New("sender not registry")
)
