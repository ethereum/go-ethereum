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
	"encoding/binary"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
)

// serveMisc is responsible for serving HTTP requests for miscellaneous data.
func (s *Service) serveMisc(w http.ResponseWriter, r *http.Request) {
	switch shift(&r.URL.Path) {
	case "cht":
		s.serveCHT(w, r)
	case "chtv2":
		s.serveCHTV2(w, r)
	case "bloomtrie":
		s.serveBloomTrie(w, r)
	}
}

// serveCHT serves the CHT request and caches the result via HTTP layer.
//
// The format of request is:
// misc/cht/<cht_root>?number=%d
func (s *Service) serveCHT(w http.ResponseWriter, r *http.Request) {
	var number uint64
	if numbers, ok := r.URL.Query()["number"]; ok {
		number, _ = strconv.ParseUint(numbers[0], 0, 64)
	}
	root, err := hexutil.Decode(shift(&r.URL.Path))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid cht root: %v", err), http.StatusBadRequest)
		return
	}
	if len(root) != common.HashLength {
		http.Error(w, fmt.Sprintf("invalid cht root: length %d != %d", len(root), common.HashLength), http.StatusBadRequest)
		return
	}
	db := s.chain.Database()
	t, err := trie.New(common.BytesToHash(root), trie.NewDatabaseWithCache(rawdb.NewTable(db, light.ChtTablePrefix), 1))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid cht trie: %v", err), http.StatusBadRequest)
	}
	// Generate merkle proof based on the user request
	proof := light.NewNodeSet()
	var key [8]byte
	binary.BigEndian.PutUint64(key[:], number)
	if err := t.Prove(key[:], 0, proof); err != nil {
		http.Error(w, fmt.Sprintf("failed to generate proof path: %v", err), http.StatusBadRequest)
	}
	replyAndCache(w, proof.NodeList()) // Done, cache it in http layer.
	log.Debug("Served cht request", "chtRoot", common.BytesToHash(root), "number", number)
}

// serveCHTV2 serves the CHT request and caches the result via HTTP layer.
//
// The format of request is:
// misc/chtv2/<tile_root>
func (s *Service) serveCHTV2(w http.ResponseWriter, r *http.Request) {
	root, err := hexutil.Decode(shift(&r.URL.Path))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid cht root: %v", err), http.StatusBadRequest)
		return
	}
	if len(root) != common.HashLength {
		http.Error(w, fmt.Sprintf("invalid cht root: length %d != %d", len(root), common.HashLength), http.StatusBadRequest)
		return
	}
	db := s.chain.Database()
	tiles, err := light.ReadCHTTile(db, common.BytesToHash(root))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid cht request: %v", err), http.StatusBadRequest)
	}
	replyAndCache(w, tiles)
}

// serveBloomTrie serves the bloomTrie request and caches the result
// via HTTP layer.
//
// The format of request is:
// misc/bloomtrie/<bloom_trie_root>?bit=%d&section=%d
func (s *Service) serveBloomTrie(w http.ResponseWriter, r *http.Request) {
	var (
		bit     uint64
		section uint64
	)
	if bits, ok := r.URL.Query()["bit"]; ok {
		bit, _ = strconv.ParseUint(bits[0], 0, 64)
	}
	if sections, ok := r.URL.Query()["section"]; ok {
		section, _ = strconv.ParseUint(sections[0], 0, 64)
	}
	root, err := hexutil.Decode(shift(&r.URL.Path))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid bloom trie root: %v", err), http.StatusBadRequest)
		return
	}
	if len(root) != common.HashLength {
		http.Error(w, fmt.Sprintf("invalid bloom trie root: length %d != %d", len(root), common.HashLength), http.StatusBadRequest)
		return
	}
	db := s.chain.Database()
	t, err := trie.New(common.BytesToHash(root), trie.NewDatabaseWithCache(rawdb.NewTable(db, light.BloomTrieTablePrefix), 1))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid bloom trie: %v", err), http.StatusBadRequest)
	}
	// Generate merkle proof based on the user request
	proof := light.NewNodeSet()
	var key [10]byte
	binary.BigEndian.PutUint16(key[:2], uint16(bit))
	binary.BigEndian.PutUint64(key[2:], section)
	if err := t.Prove(key[:], 0, proof); err != nil {
		http.Error(w, fmt.Sprintf("failed to generate proof path: %v", err), http.StatusBadRequest)
	}
	replyAndCache(w, proof.NodeList()) // Done, cache it in http layer.
	log.Debug("Served bloom trie request", "bloomRoot", common.BytesToHash(root), "bit", bit, "section", section)
}
