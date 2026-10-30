// Copyright 2026 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package posw

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/holiman/uint256"
)

func TestSimulateVerifySeal(t *testing.T) {
	for _, test := range []struct {
		name    string
		staking bool
	}{
		{name: "ordinary PoW"},
		{name: "staking", staking: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			chain, parent := newTestChain()
			config := chain.config.QuarkChain.PoSW
			config.StakingEnabled, config.WindowSize, config.DiffDivider, config.StakePerBlock = test.staking, 3, 100, big.NewInt(100)
			chain.stakes[parent.Hash()] = uint256.NewInt(100)
			engine, err := NewSimulate(chain.config.QuarkChain, parent)
			if err != nil {
				t.Fatal(err)
			}
			defer engine.Close()
			header := child(parent) // The zero Ethash mix hash is invalid.
			before := header.Hash()
			var reader consensus.ChainHeaderReader = chain
			if !test.staking {
				// Disabled staking must work without the state-reading interface.
				reader = struct{ consensus.ChainHeaderReader }{chain}
			}
			var verifier consensus.SealVerifier = engine
			if err := verifier.VerifySeal(reader, header); err != nil {
				t.Fatalf("simulated seal: %v", err)
			}
			if header.Hash() != before {
				t.Fatal("simulation modified the header")
			}
			if test.staking && chain.stakeParent != parent.Hash() {
				t.Fatal("simulation did not read the actual parent state")
			}
			if err := engine.PoSW.VerifySeal(reader, header); err == nil || err.Error() != "invalid Ethash mix hash" {
				t.Fatalf("normal PoSW must reject the invalid seal: %v", err)
			}
			header.Time = parent.Time
			if err := engine.VerifyHeader(chain, header); !errors.Is(err, errOlderBlockTime) {
				t.Fatalf("simulated header validation: have %v, want %v", err, errOlderBlockTime)
			}
		})
	}
}

func TestSimulateVerifySealErrors(t *testing.T) {
	chain, genesis := newTestChain()
	config := chain.config.QuarkChain.PoSW
	config.StakingEnabled, config.WindowSize, config.DiffDivider, config.StakePerBlock = true, 3, 100, big.NewInt(100)
	parent := child(genesis)
	chain.headers[parent.Hash()] = parent
	chain.stakes[parent.Hash()] = uint256.NewInt(100)
	header := child(parent)
	engine := &PoSWSimulate{PoSW: newTestEngine(t, chain)}

	chain.stakeErr = consensus.ErrPrunedAncestor
	if err := engine.VerifySeal(chain, header); !errors.Is(err, consensus.ErrPrunedAncestor) {
		t.Fatalf("missing parent state: have %v, want ErrPrunedAncestor", err)
	}
	chain.stakeErr = nil
	delete(chain.headers, genesis.Hash())
	if err := engine.VerifySeal(chain, header); !errors.Is(err, consensus.ErrUnknownAncestor) {
		t.Fatalf("missing history: have %v, want ErrUnknownAncestor", err)
	}
	header.MixDigest = common.Hash{}
	if err := engine.VerifySeal(chain, header); !errors.Is(err, errInvalidDifficulty) {
		t.Fatalf("zero difficulty: have %v, want %v", err, errInvalidDifficulty)
	}
}

func TestNewSimulateValidation(t *testing.T) {
	chain, genesis := newTestChain()
	if engine, err := NewSimulate(nil, genesis); err == nil || engine != nil {
		t.Fatalf("accepted missing configuration: %v, %v", engine, err)
	}
	if engine, err := NewSimulate(chain.config.QuarkChain, nil); err == nil || engine != nil {
		t.Fatalf("accepted missing genesis: %v, %v", engine, err)
	}
}
