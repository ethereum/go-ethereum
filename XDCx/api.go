package XDCx

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	LimitThresholdOrderNonceInQueue = 100
)

// List of errors
var (
	ErrNoTopics          = errors.New("missing topic(s)")
	ErrOrderNonceTooLow  = errors.New("OrderNonce too low")
	ErrOrderNonceTooHigh = errors.New("OrderNonce too high")
)

// PublicXDCXAPI provides the XDCX RPC service that can be
// use publicly without security implications.
type PublicXDCXAPI struct {
	t        *XDCX
	mu       sync.Mutex
	lastUsed map[string]time.Time // keeps track when a filter was polled for the last time.

}

// NewPublicXDCXAPI create a new RPC XDCX service.
func NewPublicXDCXAPI(t *XDCX) *PublicXDCXAPI {
	api := &PublicXDCXAPI{
		t:        t,
		lastUsed: make(map[string]time.Time),
	}
	return api
}

// Version returns the XDCX sub-protocol version.
func (api *PublicXDCXAPI) Version(ctx context.Context) string {
	return ProtocolVersionStr
}
