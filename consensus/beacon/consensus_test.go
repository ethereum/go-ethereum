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

package beacon_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// fakeChainReader implements consensus.ChainHeaderReader with just enough
// behavior for Beacon.VerifyHeader: a fixed config and a single lookup-able
// parent header. Every other method is unused by that path.
type fakeChainReader struct {
	config *params.ChainConfig
	parent *types.Header
}

func (r *fakeChainReader) Config() *params.ChainConfig  { return r.config }
func (r *fakeChainReader) CurrentHeader() *types.Header { return nil }
func (r *fakeChainReader) GetHeader(hash common.Hash, number uint64) *types.Header {
	if r.parent != nil && hash == r.parent.Hash() && number == r.parent.Number.Uint64() {
		return r.parent
	}
	return nil
}
func (r *fakeChainReader) GetHeaderByNumber(number uint64) *types.Header  { return nil }
func (r *fakeChainReader) GetHeaderByHash(hash common.Hash) *types.Header { return nil }

// TestVerifyHeaderRequestsHash checks that VerifyHeader enforces the presence
// of header.RequestsHash (EIP-7685) from Prague onward and its absence before
// Prague, mirroring the presence/absence checks already in place for
// withdrawalsHash (Shanghai) and the Amsterdam block-access-list/slotNumber
// fields.
func TestVerifyHeaderRequestsHash(t *testing.T) {
	pragueTime := uint64(1000)
	config := &params.ChainConfig{
		ChainID:      big.NewInt(1337),
		LondonBlock:  common.Big0,
		ShanghaiTime: newUint64(0),
		CancunTime:   newUint64(0),
		PragueTime:   &pragueTime,
		BlobScheduleConfig: &params.BlobScheduleConfig{
			Cancun: params.DefaultCancunBlobConfig,
			Prague: params.DefaultPragueBlobConfig,
		},
	}
	engine := beacon.New(ethash.NewFaker())

	// buildHeader returns a header at the given time that satisfies every
	// verifyHeader check except (deliberately) requestsHash, which the caller
	// sets afterwards.
	buildHeader := func(parent *types.Header, time uint64) *types.Header {
		header := &types.Header{
			ParentHash:       parent.Hash(),
			Number:           new(big.Int).Add(parent.Number, common.Big1),
			Time:             time,
			Difficulty:       common.Big0,
			Nonce:            types.EncodeNonce(0),
			UncleHash:        types.EmptyUncleHash,
			GasLimit:         parent.GasLimit,
			WithdrawalsHash:  &types.EmptyWithdrawalsHash,
			ParentBeaconRoot: &common.Hash{},
			BlobGasUsed:      new(uint64),
		}
		header.BaseFee = eip1559.CalcBaseFee(config, parent)
		excess := eip4844.CalcExcessBlobGas(config, parent, header.Time)
		header.ExcessBlobGas = &excess
		return header
	}

	genesis := &types.Header{
		Number:           common.Big0,
		Time:             0,
		Difficulty:       common.Big0,
		GasLimit:         params.MaxGasLimit / 2,
		BaseFee:          big.NewInt(params.InitialBaseFee),
		WithdrawalsHash:  &types.EmptyWithdrawalsHash,
		ParentBeaconRoot: &common.Hash{},
		ExcessBlobGas:    new(uint64),
		BlobGasUsed:      new(uint64),
	}

	verify := func(parent, header *types.Header) error {
		chain := &fakeChainReader{config: config, parent: parent}
		return engine.VerifyHeader(chain, header)
	}

	t.Run("pre-Prague header must not carry a requestsHash", func(t *testing.T) {
		header := buildHeader(genesis, pragueTime-1)
		if err := verify(genesis, header); err != nil {
			t.Fatalf("valid pre-Prague header rejected: %v", err)
		}
		bad := types.CalcRequestsHash(nil)
		header.RequestsHash = &bad
		if err := verify(genesis, header); err == nil {
			t.Fatal("pre-Prague header with a requestsHash was accepted, want rejection")
		}
	})

	t.Run("post-Prague header must carry a requestsHash", func(t *testing.T) {
		header := buildHeader(genesis, pragueTime)
		if err := verify(genesis, header); err == nil {
			t.Fatal("post-Prague header missing requestsHash was accepted, want rejection")
		}
		hash := types.CalcRequestsHash(nil)
		header.RequestsHash = &hash
		if err := verify(genesis, header); err != nil {
			t.Fatalf("valid post-Prague header rejected: %v", err)
		}
	})
}

func newUint64(n uint64) *uint64 { return &n }

var _ consensus.ChainHeaderReader = (*fakeChainReader)(nil)
