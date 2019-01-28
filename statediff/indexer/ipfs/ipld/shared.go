// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package ipld

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// IPLD Codecs for Ethereum
// See the authoritative document:
// https://github.com/multiformats/multicodec/blob/master/table.csv
const (
	RawBinary           = 0x55
	MEthHeader          = 0x90
	MEthHeaderList      = 0x91
	MEthTxTrie          = 0x92
	MEthTx              = 0x93
	MEthTxReceiptTrie   = 0x94
	MEthTxReceipt       = 0x95
	MEthStateTrie       = 0x96
	MEthAccountSnapshot = 0x97
	MEthStorageTrie     = 0x98
)

// RawdataToCid takes the desired codec and a slice of bytes
// and returns the proper cid of the object.
func RawdataToCid(codec uint64, rawdata []byte, multiHash uint64) (cid.Cid, error) {
	c, err := cid.Prefix{
		Codec:    codec,
		Version:  1,
		MhType:   multiHash,
		MhLength: -1,
	}.Sum(rawdata)
	if err != nil {
		return cid.Cid{}, err
	}
	return c, nil
}

// keccak256ToCid takes a keccak256 hash and returns its cid based on
// the codec given.
func keccak256ToCid(codec uint64, h []byte) cid.Cid {
	buf, err := mh.Encode(h, mh.KECCAK_256)
	if err != nil {
		panic(err)
	}

	return cid.NewCidV1(codec, mh.Multihash(buf))
}

// commonHashToCid takes a go-ethereum common.Hash and returns its
// cid based on the codec given,
func commonHashToCid(codec uint64, h common.Hash) cid.Cid {
	mhash, err := mh.Encode(h[:], mh.KECCAK_256)
	if err != nil {
		panic(err)
	}

	return cid.NewCidV1(codec, mhash)
}

// localTrie wraps a go-ethereum trie and its underlying memory db.
// It contributes to the creation of the trie node objects.
type localTrie struct {
	keys [][]byte
	db   ethdb.Database
	trie *trie.Trie
}

// newLocalTrie initializes and returns a localTrie object
func newLocalTrie() *localTrie {
	var err error
	lt := &localTrie{}
	lt.db = rawdb.NewMemoryDatabase()
	lt.trie, err = trie.New(common.Hash{}, trie.NewDatabase(lt.db))
	if err != nil {
		panic(err)
	}
	return lt
}

// add receives the index of an object and its rawdata value
// and includes it into the localTrie
func (lt *localTrie) add(idx int, rawdata []byte) {
	key, err := rlp.EncodeToBytes(uint(idx))
	if err != nil {
		panic(err)
	}
	lt.keys = append(lt.keys, key)
	if err := lt.db.Put(key, rawdata); err != nil {
		panic(err)
	}
	lt.trie.Update(key, rawdata)
}

// rootHash returns the computed trie root.
// Useful for sanity checks on parsed data.
func (lt *localTrie) rootHash() []byte {
	return lt.trie.Hash().Bytes()
}

// getKeys returns the stored keys of the memory database
// of the localTrie for further processing.
func (lt *localTrie) getKeys() [][]byte {
	return lt.keys
}
