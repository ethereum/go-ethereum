// Copyright 2026-2027 The quarkchain Authors
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

package txpool

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// TestValidateTransactionRejectsDeposit checks that no pool accepts cross-shard
// deposits, which only the consensus layer may include in a block.
func TestValidateTransactionRejectsDeposit(t *testing.T) {
	var (
		config = *params.MergedTestChainConfig
		head   = &types.Header{Number: big.NewInt(1), Difficulty: common.Big0}
		tx     = types.NewTx(&types.DepositTx{Value: uint256.NewInt(1), GasPrice: uint256.NewInt(1), Gas: params.XShardSendGas})
	)
	config.QuarkChain = &params.QuarkChainConfig{ShardID: 1, ShardIDs: []uint32{0, 1}}
	opts := &ValidationOptions{Config: &config, Accept: 0xff, MaxSize: 1 << 20, MinTip: new(big.Int)}
	if err := ValidateTransaction(tx, head, types.LatestSigner(&config), opts); !errors.Is(err, core.ErrTxTypeNotSupported) {
		t.Errorf("deposit validation error %v, want %v", err, core.ErrTxTypeNotSupported)
	}
}
