// Copyright 2014 The go-ethereum Authors
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

package state

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// DumpConfig is a set of options to control what portions of the state will be
// iterated and collected.
type DumpConfig struct {
	SkipCode          bool
	SkipStorage       bool
	OnlyWithAddresses bool
	Start             []byte
	Max               uint64
}

// DumpCollector interface which the state trie calls during iteration
type DumpCollector interface {
	// OnRoot is called with the state root
	OnRoot(common.Hash)
	// OnAccount is called once for each account in the trie
	OnAccount(*common.Address, DumpAccount)
}

// DumpAccount represents an account in the state.
type DumpAccount struct {
	Balance     string                 `json:"balance"`
	Nonce       uint64                 `json:"nonce"`
	Root        hexutil.Bytes          `json:"root"`
	CodeHash    hexutil.Bytes          `json:"codeHash"`
	Code        hexutil.Bytes          `json:"code,omitempty"`
	Storage     map[common.Hash]string `json:"storage,omitempty"`
	Address     *common.Address        `json:"address,omitempty"` // Address only present in iterative (line-by-line) mode
	AddressHash hexutil.Bytes          `json:"key,omitempty"`     // If we don't have address, we can output the key
}

// Dump represents the full dump in a collected format, as one large map.
type Dump struct {
	Root     string                 `json:"root"`
	Accounts map[string]DumpAccount `json:"accounts"`
	// Next can be set to represent that this dump is only partial, and Next
	// is where an iterator should be positioned in order to continue the dump.
	Next []byte `json:"next,omitempty"` // nil if no more accounts
}

// OnRoot implements DumpCollector interface
func (d *Dump) OnRoot(root common.Hash) {
	d.Root = fmt.Sprintf("%x", root)
}

// OnAccount implements DumpCollector interface
func (d *Dump) OnAccount(addr *common.Address, account DumpAccount) {
	if addr == nil {
		d.Accounts[fmt.Sprintf("pre(%s)", account.AddressHash)] = account
	}
	if addr != nil {
		d.Accounts[(*addr).String()] = account
	}
}

// iterativeDump is a DumpCollector-implementation which dumps output line-by-line iteratively.
type iterativeDump struct {
	*json.Encoder
}

// OnAccount implements DumpCollector interface
func (d iterativeDump) OnAccount(addr *common.Address, account DumpAccount) {
	dumpAccount := &DumpAccount{
		Balance:     account.Balance,
		Nonce:       account.Nonce,
		Root:        account.Root,
		CodeHash:    account.CodeHash,
		Code:        account.Code,
		Storage:     account.Storage,
		AddressHash: account.AddressHash,
		Address:     addr,
	}
	d.Encode(dumpAccount)
}

// OnRoot implements DumpCollector interface
func (d iterativeDump) OnRoot(root common.Hash) {
	d.Encode(struct {
		Root common.Hash `json:"root"`
	}{root})
}

// DumpToCollector iterates the state according to the given options and inserts
// the items into a collector for aggregation or serialization.
func (s *StateDB) DumpToCollector(c DumpCollector, conf *DumpConfig) (nextKey []byte) {
	// Sanitize the input to allow nil configs
	if conf == nil {
		conf = new(DumpConfig)
	}
	var (
		missingPreimages int
		accounts         uint64
		start            = time.Now()
		logged           = time.Now()
	)
	log.Info("Trie dumping started", "root", s.originalRoot)
	c.OnRoot(s.originalRoot)

	iteratee, err := s.db.Iteratee(s.originalRoot)
	if err != nil {
		return nil
	}
	var startHash common.Hash
	if conf.Start != nil {
		startHash = common.BytesToHash(conf.Start)
	}
	acctIt, err := iteratee.NewAccountIterator(startHash)
	if err != nil {
		return nil
	}
	defer acctIt.Release()

	for acctIt.Next() {
		var data types.StateAccount
		if err := rlp.DecodeBytes(acctIt.Account(), &data); err != nil {
			panic(err)
		}
		var (
			account = DumpAccount{
				Balance:     data.Balance.String(),
				Nonce:       data.Nonce,
				Root:        data.Root[:],
				CodeHash:    data.CodeHash,
				AddressHash: acctIt.Hash().Bytes(),
			}
			address *common.Address
			addr    common.Address
		)
		addrBytes, err := acctIt.Address()
		if err != nil {
			missingPreimages++
			if conf.OnlyWithAddresses {
				continue
			}
		} else {
			address = &addrBytes
			account.Address = address
		}
		obj := newObject(s, addr, &data)
		if !conf.SkipCode {
			account.Code = obj.Code()
		}
		if !conf.SkipStorage {
			account.Storage = make(map[common.Hash]string)

			storageIt, err := iteratee.NewStorageIterator(acctIt.Hash(), obj.Root(), common.Hash{})
			if err != nil {
				log.Error("Failed to load storage trie", "err", err)
				continue
			}

			for storageIt.Next() {
				_, content, _, err := rlp.Split(storageIt.Slot())
				if err != nil {
					log.Error("Failed to decode the value returned by iterator", "error", err)
					continue
				}
				key, err := storageIt.Key()
				if err != nil {
					continue
				}
				account.Storage[key] = common.Bytes2Hex(content)
			}
			storageIt.Release()
		}
		c.OnAccount(address, account)
		accounts++
		if time.Since(logged) > 8*time.Second {
			log.Info("Trie dumping in progress", "at", acctIt.Hash().Hex(), "accounts", accounts,
				"elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
		if conf.Max > 0 && accounts >= conf.Max {
			if acctIt.Next() {
				nextKey = acctIt.Hash().Bytes()
			}
			break
		}
	}
	if missingPreimages > 0 {
		log.Warn("Dump incomplete due to missing preimages", "missing", missingPreimages)
	}
	log.Info("Trie dumping complete", "accounts", accounts,
		"elapsed", common.PrettyDuration(time.Since(start)))

	return nextKey
}

// RawDump returns the state. If the processing is aborted e.g. due to options
// reaching Max, the `Next` key is set on the returned Dump.
func (s *StateDB) RawDump(opts *DumpConfig) Dump {
	dump := &Dump{
		Accounts: make(map[string]DumpAccount),
	}
	dump.Next = s.DumpToCollector(dump, opts)
	return *dump
}

// Dump returns a JSON string representing the entire state as a single json-object
func (s *StateDB) Dump(opts *DumpConfig) []byte {
	dump := s.RawDump(opts)
	json, err := json.MarshalIndent(dump, "", "    ")
	if err != nil {
		log.Error("Error dumping state", "err", err)
	}
	return json
}

// IterativeDump dumps out accounts as json-objects, delimited by linebreaks on stdout
func (s *StateDB) IterativeDump(opts *DumpConfig, output *json.Encoder) {
	s.DumpToCollector(iterativeDump{output}, opts)
}
