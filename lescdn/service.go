package lescdn

import (
	"net/http"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

// Service is a RESTful HTTP service meant to act as a data source for a light client
// content distribution network.
type Service struct {
	chain *core.BlockChain
}

// New creates a data source for a les content distribution network.
func New(chain *core.BlockChain) *Service {
	return &Service{
		chain: chain,
	}
}

// Protocols implements node.Service, returning the P2P network protocols used
// by the lescdn service (nil as it doesn't use the devp2p overlay network).
func (s *Service) Protocols() []p2p.Protocol { return nil }

// APIs implements node.Service, returning the RPC API endpoints provided by the
// lescdn service (nil as it doesn't provide any user callable APIs).
func (s *Service) APIs() []rpc.API { return nil }

// Start implements node.Service, starting up the content distribution source.
func (s *Service) Start(server *p2p.Server) error {
	go http.ListenAndServe("localhost:8548", newGzipHandler(s))

	log.Info("Light client CDN started")
	return nil
}

// Stop implements node.Service, terminating the content distribution source.
func (s *Service) Stop() error {
	log.Info("Light client CDN stopped")
	return nil
}

// ServeHTTP is the entry point of the les cdn, splitting the request across the
// supported submodules.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch shift(&r.URL.Path) {
	case "chain":
		s.serveChain(w, r)
		return

	case "state":
		s.serveState(w, r)
		return
	}
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

// shift splits off the first component of p, which will be cleaned of relative
// components before processing. The returned head will never contain a slash and
// the remaining tail will always be a rooted path without trailing slash.
func shift(p *string) string {
	*p = path.Clean("/" + *p)

	var head string
	if idx := strings.Index((*p)[1:], "/") + 1; idx > 0 {
		head = (*p)[1:idx]
		*p = (*p)[idx:]
	} else {
		head = (*p)[1:]
		*p = "/"
	}
	return head
}

// reply marshals a value into the response stream via RLP, also setting caching
// to indefinite.
func reply(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Cache-Control", "max-age=31536000") // 1 year cache expiry
	if err := rlp.Encode(w, v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
