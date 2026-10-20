// Copyright 2021 The go-ethereum Authors
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

package eip1559

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// copyConfig does a _shallow_ copy of a given config. Safe to set new values, but
// do not use e.g. SetInt() on the numbers. For testing only
func copyConfig(original *params.ChainConfig) *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:                 original.ChainID,
		HomesteadBlock:          original.HomesteadBlock,
		DAOForkBlock:            original.DAOForkBlock,
		DAOForkSupport:          original.DAOForkSupport,
		EIP150Block:             original.EIP150Block,
		EIP155Block:             original.EIP155Block,
		EIP158Block:             original.EIP158Block,
		ByzantiumBlock:          original.ByzantiumBlock,
		ConstantinopleBlock:     original.ConstantinopleBlock,
		PetersburgBlock:         original.PetersburgBlock,
		IstanbulBlock:           original.IstanbulBlock,
		MuirGlacierBlock:        original.MuirGlacierBlock,
		BerlinBlock:             original.BerlinBlock,
		LondonBlock:             original.LondonBlock,
		TerminalTotalDifficulty: original.TerminalTotalDifficulty,
		Ethash:                  original.Ethash,
		Clique:                  original.Clique,
	}
}

func config() *params.ChainConfig {
	config := copyConfig(params.TestChainConfig)
	config.LondonBlock = big.NewInt(5)
	return config
}

// TestBlockGasLimits tests the gasLimit checks for blocks both across
// the EIP-1559 boundary and post-1559 blocks
func TestBlockGasLimits(t *testing.T) {
	initial := new(big.Int).SetUint64(params.InitialBaseFee)

	for i, tc := range []struct {
		pGasLimit uint64
		pNum      int64
		gasLimit  uint64
		ok        bool
	}{
		// Transitions from non-london to london
		{10000000, 4, 20000000, true},  // No change
		{10000000, 4, 20019530, true},  // Upper limit
		{10000000, 4, 20019531, false}, // Upper +1
		{10000000, 4, 19980470, true},  // Lower limit
		{10000000, 4, 19980469, false}, // Lower limit -1
		// London to London
		{20000000, 5, 20000000, true},
		{20000000, 5, 20019530, true},  // Upper limit
		{20000000, 5, 20019531, false}, // Upper limit +1
		{20000000, 5, 19980470, true},  // Lower limit
		{20000000, 5, 19980469, false}, // Lower limit -1
		{40000000, 5, 40039061, true},  // Upper limit
		{40000000, 5, 40039062, false}, // Upper limit +1
		{40000000, 5, 39960939, true},  // lower limit
		{40000000, 5, 39960938, false}, // Lower limit -1
	} {
		parent := &types.Header{
			GasUsed:  tc.pGasLimit / 2,
			GasLimit: tc.pGasLimit,
			BaseFee:  initial,
			Number:   big.NewInt(tc.pNum),
		}
		header := &types.Header{
			GasUsed:  tc.gasLimit / 2,
			GasLimit: tc.gasLimit,
			BaseFee:  initial,
			Number:   big.NewInt(tc.pNum + 1),
		}
		err := VerifyEIP1559Header(config(), parent, header)
		if tc.ok && err != nil {
			t.Errorf("test %d: Expected valid header: %s", i, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("test %d: Expected invalid header", i)
		}
	}
}

// TestCalcBaseFee assumes all blocks are 1559-blocks
func TestCalcBaseFee(t *testing.T) {
	tests := []struct {
		parentBaseFee   int64
		parentGasLimit  uint64
		parentGasUsed   uint64
		expectedBaseFee int64
	}{
		{params.InitialBaseFee, 20000000, 10000000, params.InitialBaseFee}, // usage == target
		{params.InitialBaseFee, 20000000, 9000000, 987500000},              // usage below target
		{params.InitialBaseFee, 20000000, 11000000, 1012500000},            // usage above target
	}
	for i, test := range tests {
		parent := &types.Header{
			Number:   common.Big32,
			GasLimit: test.parentGasLimit,
			GasUsed:  test.parentGasUsed,
			BaseFee:  big.NewInt(test.parentBaseFee),
		}
		if have, want := CalcBaseFee(config(), parent, &types.Header{}), big.NewInt(test.expectedBaseFee); have.Cmp(want) != 0 {
			t.Errorf("test %d: have %d  want %d, ", i, have, want)
		}
	}
}

// TestCalcBaseFeeBogota checks the EIP-8198 max change of seconds per slot over
// 96, applied after dividing by the gas target like the spec does.
func TestCalcBaseFeeBogota(t *testing.T) {
	cfg := config()
	cfg.BogotaTime = new(uint64)

	tests := []struct {
		parentBaseFee   int64
		parentGasUsed   uint64
		seconds, slots  uint64
		expectedBaseFee int64
	}{
		{params.InitialBaseFee, 9000000, 10, 1, 989583334},     // 10s slots, usage below target
		{params.InitialBaseFee, 11000000, 10, 1, 1010416666},   // 10s slots, usage above target
		{params.InitialBaseFee, 11000000, 12, 1, 1012500000},   // 12s slots keep the 1/8 rule
		{params.InitialBaseFee, 11000000, 20, 2, 1010416666},   // a missed 10s slot keeps 10s per slot
		{params.InitialBaseFee, 20000000, 1000, 1, 2000000000}, // capped at the whole base fee
		{97000000, 9999999, 10, 1, 97000000},                   // multiplying by the seconds first would decrease by 1
		{195000000, 10000001, 10, 1, 195000001},                // multiplying by the seconds first would increase by 2
	}
	for i, test := range tests {
		var (
			parentSlot = uint64(100)
			slot       = parentSlot + test.slots
			parent     = &types.Header{
				Number:     common.Big32,
				Time:       1000,
				SlotNumber: &parentSlot,
				GasLimit:   20000000,
				GasUsed:    test.parentGasUsed,
				BaseFee:    big.NewInt(test.parentBaseFee),
			}
			header = &types.Header{Time: parent.Time + test.seconds, SlotNumber: &slot}
		)
		if have, want := CalcBaseFee(cfg, parent, header), big.NewInt(test.expectedBaseFee); have.Cmp(want) != 0 {
			t.Errorf("test %d: have %d  want %d, ", i, have, want)
		}
	}
}

// TestBaseFeeBogotaTransition checks the base fee and slot number rules on both
// sides of the fork, through header verification.
func TestBaseFeeBogotaTransition(t *testing.T) {
	cfg := config()
	bogotaTime := uint64(1000)
	cfg.BogotaTime = &bogotaTime

	for i, tc := range []struct {
		parentTime, time uint64
		slot             uint64
		baseFee          int64
		wantErr          string
	}{
		{988, 999, 101, 1125000000, ""},                    // 1/8 before the fork, whatever the spacing
		{988, 1000, 101, 1125000000, ""},                   // first Bogota block, 12s after its parent
		{990, 1000, 101, 1104166666, ""},                   // 10s after its parent gives 5/48
		{990, 1000, 100, 1104166666, "invalid slotNumber"}, // the slot number must advance
	} {
		var (
			parentSlot = uint64(100)
			slot       = tc.slot
			parent     = &types.Header{
				Number:     common.Big32,
				Time:       tc.parentTime,
				SlotNumber: &parentSlot,
				GasLimit:   20000000,
				GasUsed:    20000000,
				BaseFee:    big.NewInt(params.InitialBaseFee),
			}
			header = &types.Header{
				Number:     big.NewInt(33),
				Time:       tc.time,
				SlotNumber: &slot,
				GasLimit:   parent.GasLimit,
				BaseFee:    big.NewInt(tc.baseFee),
			}
		)
		err := VerifyEIP1559Header(cfg, parent, header)
		switch {
		case tc.wantErr == "" && err != nil:
			t.Errorf("test %d: expected valid header: %v", i, err)
		case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
			t.Errorf("test %d: expected %q error, got %v", i, tc.wantErr, err)
		}
	}
}

// TestCalcNextBaseFee checks that the next-block estimate assumes one slot at the
// reference duration, since the real spacing isn't known yet.
func TestCalcNextBaseFee(t *testing.T) {
	cfg := config()
	cfg.BogotaTime = new(uint64)

	slot := uint64(100)
	head := &types.Header{
		Number:     common.Big32,
		Time:       1000,
		SlotNumber: &slot,
		GasLimit:   20000000,
		GasUsed:    20000000,
		BaseFee:    big.NewInt(params.InitialBaseFee),
	}
	if have, want := CalcNextBaseFee(cfg, head), big.NewInt(1125000000); have.Cmp(want) != 0 {
		t.Errorf("have %d  want %d", have, want)
	}
}
