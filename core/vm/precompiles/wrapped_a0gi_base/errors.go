package wrappeda0gibase

import "errors"

var (
	ErrSenderNotWA0GI         = errors.New("sender is not WA0GI")
	ErrSenderNotAgency        = errors.New("sender is not agency")
	ErrInsufficientMintCap    = errors.New("insufficient mint cap")
	ErrInsufficientMintSupply = errors.New("insufficient mint supply")
)
