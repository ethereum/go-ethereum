package XDCxlending

import (
	"context"
	"errors"
	"sync"
	"time"
)

// List of errors
var (
	ErrOrderNonceTooLow  = errors.New("OrderNonce too low")
	ErrOrderNonceTooHigh = errors.New("OrderNonce too high")
)

// PublicXDCXLendingAPI provides the XDCX RPC service that can be
// use publicly without security implications.
type PublicXDCXLendingAPI struct {
	t        *Lending
	mu       sync.Mutex
	lastUsed map[string]time.Time // keeps track when a filter was polled for the last time.

}

// NewPublicXDCXLendingAPI create a new RPC XDCX service.
func NewPublicXDCXLendingAPI(t *Lending) *PublicXDCXLendingAPI {
	api := &PublicXDCXLendingAPI{
		t:        t,
		lastUsed: make(map[string]time.Time),
	}
	return api
}

// Version returns the Lending sub-protocol version.
func (api *PublicXDCXLendingAPI) Version(ctx context.Context) string {
	return ProtocolVersionStr
}
