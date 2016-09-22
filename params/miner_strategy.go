// Copyright 2016 The go-ethereum Authors
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

package params

import (
	"math/big"
	"sync"
	"time"
)

var (
	GenesisGasLimit = big.NewInt(4712388)               // Gas limit of the Genesis block.
	TargetGasLimit  = new(big.Int).Set(GenesisGasLimit) // The artificial target
	BlockTimeLimit  = 5 * time.Second                   // Block processing time limit to reduce gas after

	// Temp hack to get a dynamic gas limit in palace (clean up!!!)
	CurrentGasCeil       = new(big.Int).Set(GenesisGasLimit)
	CurrentGasCeilCutDiv = big.NewInt(2)
	CurrentGasCeilIncDiv = new(big.Int).Set(GasLimitBoundDivisor)
	CurrentGasCeilLock   sync.Mutex
)
