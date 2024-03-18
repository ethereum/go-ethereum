// Copyright 2024 The go-ethereum Authors
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

package triedb

import (
	"errors"

	"github.com/ethereum/go-ethereum/triedb/hashdb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// Config defines all options for configuring database.
type Config struct {
	Preimages bool           // Flag whether the preimage of node key is recorded
	IsVerkle  bool           // Flag whether the db is holding a verkle tree
	HashDB    *hashdb.Config // Configs for hash-based scheme
	PathDB    *pathdb.Config // Configs for experimental path-based scheme
}

// sanitize validates the provided config.
func (config *Config) sanitize() error {
	if config == nil {
		return errors.New("config is nil")
	}
	if config.HashDB != nil && config.PathDB != nil {
		return errors.New("both 'hash' and 'path' mode are configured")
	}
	if config.HashDB == nil && config.PathDB == nil {
		return errors.New("neither 'hash' nor 'path' mode is configured")
	}
	return nil
}

// Copy returns a deep copied config object.
func (config *Config) Copy() *Config {
	cpy := &Config{
		Preimages: config.Preimages,
		IsVerkle:  config.IsVerkle,
	}
	if config.HashDB != nil {
		cpy.HashDB = config.HashDB.Copy()
	}
	if config.PathDB != nil {
		cpy.PathDB = config.PathDB.Copy()
	}
	return cpy
}
