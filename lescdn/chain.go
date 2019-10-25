// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package lescdn

import (
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
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
				replyAndCache(w, header)
				log.Debug("Served chain item", "type", "header", "hash", hash)
				return
			}
			// Header not found, error out appropriately
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return

		case "uncles":
			// Retrieve the block and attempt to return the uncles
			if block := s.chain.GetBlockByHash(hash); block != nil {
				replyAndCache(w, block.Uncles())
				log.Debug("Served chain item", "type", "uncles", "hash", hash)
				return
			}
			// Block not found, error out appropriately
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return

		case "transactions":
			// Retrieve the block and attempt to return the transactions
			if block := s.chain.GetBlockByHash(hash); block != nil {
				replyAndCache(w, block.Transactions())
				log.Debug("Served chain item", "type", "transactions", "hash", hash)
				return
			}
			// Block not found, error out appropriately
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return

		case "receipts":
			// Retrieve the receipts and attempt to return them
			if receipts := s.chain.GetReceiptsByHash(hash); receipts != nil {
				replyAndCache(w, receipts)
				log.Debug("Served chain item", "type", "receipts", "hash", hash)
				return
			}
			// Receipts not found, error out appropriately
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return

		case "txstatus":
			// Retrieve the tx lookup and attempt to return them
			if lookup := s.chain.GetTransactionLookup(hash); lookup != nil {
				reply(w, lookup) // Can't cache in HTTP layer, since it's still mutable.
				log.Debug("Served chain item", "type", "txstatus", "hash", hash)
				return
			}
			// Tx lookup not found, error out appropriately
			// Note in theory we should check txpool for the pending transaction,
			// But it will increase lots of pressure to txpool, and also the status
			// of transaction in pool is not suitable to cache in CDN.
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	})
}
