package health

import (
	"net/http"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
)

type handler struct {
	ec *ethclient.Client
}

// ServeHTTP implements the http.Handler interface.
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	headers := r.Header.Values(healthHeader)
	if len(headers) != 0 {
		processFromHeaders(h.ec, headers, w, r)
	} else {
		processFromBody(h.ec, w, r)
	}
}

// New constructs a new health service instance.
func New(stack *node.Node, cors, vhosts []string) error {
	_, err := newHandler(stack, cors, vhosts)
	return err
}

// newHandler returns a new `http.Handler` that will answer node health queries.
func newHandler(stack *node.Node, cors, vhosts []string) (*handler, error) {
	ec := ethclient.NewClient(stack.Attach())
	h := handler{ec}
	handler := node.NewHTTPHandlerStack(h, cors, vhosts, nil)

	stack.RegisterHandler("Health API", "/health", handler)
	stack.RegisterHandler("Health API", "/health/", handler)

	return &h, nil
}
