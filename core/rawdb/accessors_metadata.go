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
	"fmt"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

// ReadDatabaseVersion retrieves the version number of the database.
func ReadDatabaseVersion(db ethdb.KeyValueReader) *uint64 {
	var version uint64

	enc, _ := db.Get(databaseVersionKey)
	if len(enc) == 0 {
		return nil
	}
	if err := rlp.DecodeBytes(enc, &version); err != nil {
		return nil
	}

	return &version
}

// WriteDatabaseVersion stores the version number of the database
func WriteDatabaseVersion(db ethdb.KeyValueWriter, version uint64) {
	enc, err := rlp.EncodeToBytes(version)
	if err != nil {
		log.Crit("Failed to encode database version", "err", err)
	}
	if err = db.Put(databaseVersionKey, enc); err != nil {
		log.Crit("Failed to store the database version", "err", err)
	}
}

// ReadChainConfig will fetch the network settings based on the given hash.
func ReadChainConfig(db ethdb.KeyValueReader, hash common.Hash) (*params.ChainConfig, error) {
	jsonChainConfig, _ := db.Get(configKey(hash))
	if len(jsonChainConfig) == 0 {
		return nil, errors.New("ChainConfig not found") // general config not found error
	}

	var config params.ChainConfig
	if err := json.Unmarshal(jsonChainConfig, &config); err != nil {
		log.Error("Invalid chain config JSON", "hash", hash, "err", err)
		return nil, fmt.Errorf("invalid chain config JSON for hash %s: %w", hash.Hex(), err)
	}

	return &config, nil
}

// WriteChainConfig writes the chain config settings to the database.
func WriteChainConfig(db ethdb.KeyValueWriter, hash common.Hash, cfg *params.ChainConfig) {
	if cfg == nil {
		return
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		log.Crit("Failed to JSON encode chain config", "err", err)
	}
	if err := db.Put(configKey(hash), data); err != nil {
		log.Crit("Failed to store chain config", "err", err)
	}
}

// ReadGenesisStateSpec retrieves the genesis state specification based on the
// given genesis (block-)hash.
func ReadGenesisStateSpec(db ethdb.KeyValueReader, blockhash common.Hash) []byte {
	data, _ := db.Get(genesisStateSpecKey(blockhash))
	return data
}

// WriteGenesisStateSpec writes the genesis state specification into the disk.
func WriteGenesisStateSpec(db ethdb.KeyValueWriter, blockhash common.Hash, data []byte) {
	if err := db.Put(genesisStateSpecKey(blockhash), data); err != nil {
		log.Crit("Failed to store genesis state", "err", err)
	}
}
