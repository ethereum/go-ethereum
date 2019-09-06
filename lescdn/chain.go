package lescdn

import (
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// serveChain is responsible for serving HTTP requests for chain data.
func (s *Service) serveChain(w http.ResponseWriter, r *http.Request) {
	hash, err := hexutil.Decode(shift(&r.URL.Path))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid block hash: %v", err), http.StatusBadRequest)
		return
	}
	if len(hash) != common.HashLength {
		http.Error(w, fmt.Sprintf("invalid block hash: length %d != %d", len(hash), common.HashLength), http.StatusBadRequest)
		return
	}
	s.serveChainItem(common.BytesToHash(hash)).ServeHTTP(w, r)
}

// serveChainItem is responsible for creating a server for HTTP requests for some
// component of a chain item.
func (s *Service) serveChainItem(hash common.Hash) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch shift(&r.URL.Path) {
		case "header":
			// Retrieve the header and attempt to return it
			if header := s.chain.GetHeaderByHash(hash); header != nil {
				reply(w, header)
				return
			}
			// Header not found, error out appropriately
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return

		case "uncles":
			// Retrieve the block and attempt to return the uncles
			if block := s.chain.GetBlockByHash(hash); block != nil {
				reply(w, block.Uncles())
				return
			}
			// Block not found, error out appropriately
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return

		case "transactions":
			// Retrieve the block and attempt to return the transactions
			if block := s.chain.GetBlockByHash(hash); block != nil {
				reply(w, block.Transactions())
				return
			}
			// Block not found, error out appropriately
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return

		case "receipts":
			// Retrieve the receipts and attempt to return them
			if receipts := s.chain.GetReceiptsByHash(hash); receipts != nil {
				reply(w, receipts)
				return
			}
			// Receipts not found, error out appropriately
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	})
}
