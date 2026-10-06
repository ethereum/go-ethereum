// Copyright 2026 The go-ethereum Authors
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

package framepool

import "github.com/ethereum/go-ethereum/log"

const (
	txSlotSize = 32 * 1024
	txMaxSize  = 4 * txSlotSize
)

// Config contains the resource limits of the frame transaction pool.
type Config struct {
	GlobalSlots uint64 // Maximum number of 32 KiB transaction slots
	PriceBump   uint64 // Minimum percentage increase of both replacement fees
}

// DefaultConfig contains the default frame pool limits.
var DefaultConfig = Config{
	GlobalSlots: 1024,
	PriceBump:   10,
}

func (config *Config) sanitize() Config {
	conf := *config
	if conf.GlobalSlots < 1 {
		log.Warn("Sanitizing invalid framepool slot cap", "provided", conf.GlobalSlots, "updated", DefaultConfig.GlobalSlots)
		conf.GlobalSlots = DefaultConfig.GlobalSlots
	}
	if conf.PriceBump < 1 {
		log.Warn("Sanitizing invalid framepool price bump", "provided", conf.PriceBump, "updated", DefaultConfig.PriceBump)
		conf.PriceBump = DefaultConfig.PriceBump
	}
	return conf
}
