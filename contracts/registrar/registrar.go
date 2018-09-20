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

package registrar

//go:generate abigen --sol contract/registrar.sol --pkg contract --out contract/registrar.go

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/registrar/contract"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type Registrar struct {
	contract *contract.Contract
}

// NewRegistrar binds checkpoint contract and returns a registrar instance.
func NewRegistrar(contractAddr common.Address, backend bind.ContractBackend) (*Registrar, error) {
	c, err := contract.NewContract(contractAddr, backend)
	if err != nil {
		return nil, err
	}
	return &Registrar{contract: c}, nil
}

// Contract returns the underlying contract instance.
func (registrar *Registrar) Contract() *contract.Contract {
	return registrar.contract
}

// LookupCheckpointEvent searches checkpoint event for specific section in the given log batches.
func (registrar *Registrar) LookupCheckpointEvent(blockLogs [][]*types.Log, section uint64, hash common.Hash) []*contract.ContractNewCheckpointEvent {
	var result []*contract.ContractNewCheckpointEvent

	for _, logs := range blockLogs {
		for _, log := range logs {
			event, err := registrar.contract.ParseNewCheckpointEvent(*log)
			if err != nil {
				continue
			}
			if event.Index.Uint64() == section && common.Hash(event.CheckpointHash) == hash {
				result = append(result, event)
			}
		}
	}
	return result
}

// SetCheckpoint creates a signature for given checkpoint with specified private key and registers into contract.
func (registrar *Registrar) SetCheckpoint(key *ecdsa.PrivateKey, sectionIndex *big.Int, hash []byte) (*types.Transaction, error) {
	sig, err := crypto.Sign(hash, key)
	if err != nil {
		return nil, err
	}
	var h [32]byte
	copy(h[:], hash)
	return registrar.contract.SetCheckpoint(bind.NewKeyedTransactor(key), sectionIndex, h, sig)
}
