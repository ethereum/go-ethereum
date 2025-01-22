// Copyright 2018 The go-ethereum Authors
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

package rawdb

import (
	"encoding/json"
	"errors"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

// ReadDatabaseVersion reads the version number from db.
func ReadDatabaseVersion(db ethdb.KeyValueReader) int {
	var vsn uint
	enc, _ := db.Get(databaseVersionKey)
	rlp.DecodeBytes(enc, &vsn)
	return int(vsn)
}

// WriteDatabaseVersion writes vsn as the version number to db.
func WriteDatabaseVersion(db ethdb.KeyValueWriter, vsn int) {
	enc, _ := rlp.EncodeToBytes(uint(vsn))
	db.Put(databaseVersionKey, enc)
}

// ReadChainConfig will fetch the network settings based on the given hash.
func ReadChainConfig(db ethdb.KeyValueReader, hash common.Hash) (*params.ChainConfig, error) {
	jsonChainConfig, _ := db.Get(configKey(hash))
	if len(jsonChainConfig) == 0 {
		return nil, errors.New("ChainConfig not found") // general config not found error
	}

	var config params.ChainConfig
	if err := json.Unmarshal(jsonChainConfig, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// WriteChainConfig writes the chain config settings to the database.
func WriteChainConfig(db ethdb.KeyValueWriter, hash common.Hash, cfg *params.ChainConfig) error {
	// short circuit and ignore if nil config. GetChainConfig
	// will return a default.
	if cfg == nil {
		return nil
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		log.Crit("Failed to JSON encode chain config", "err", err)
		return err
	}
	return db.Put(configKey(hash), data)
}

// ReadPreimage returns a Database instance with the key prefix for preimage entries.
func ReadPreimage(db ethdb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(preimageKey(hash))
	return data
}

// WritePreimages writes the provided set of preimages to the database.
func WritePreimages(db ethdb.KeyValueWriter, preimages map[common.Hash][]byte) {
	for hash, preimage := range preimages {
		if err := db.Put(preimageKey(hash), preimage); err != nil {
			log.Crit("Failed to store trie preimage", "err", err)
		}
	}
	preimageCounter.Inc(int64(len(preimages)))
	preimageHitCounter.Inc(int64(len(preimages)))
}
