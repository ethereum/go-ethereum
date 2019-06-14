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

// Package registrar is a an on-chain light client checkpoint oracle.
package registrar

//go:generate abigen --sol contract/registrar.sol --pkg contract --out contract/registrar.go

import (
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/registrar/contract"
	"github.com/ethereum/go-ethereum/core/types"
)

// Registrar is a Go wrapper around an on-chain light client checkpoint oracle.
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

// LookupCheckpointEvents searches checkpoint event for specific section in the
// given log batches.
func (registrar *Registrar) LookupCheckpointEvents(blockLogs [][]*types.Log, section uint64, hash common.Hash) []*contract.ContractNewCheckpointVote {
	var votes []*contract.ContractNewCheckpointVote

	for _, logs := range blockLogs {
		for _, log := range logs {
			event, err := registrar.contract.ParseNewCheckpointVote(*log)
			if err != nil {
				continue
			}
			if event.Index == section && common.Hash(event.CheckpointHash) == hash {
				votes = append(votes, event)
			}
		}
	}
	return votes
}

// RegisterCheckpoint registers the checkpoint with a batch of associated signatures
// that are collected off-chain and sorted by lexicographical order.
//
// Notably all signatures given should be transformed to "ethereum style" which transforms
// v from 0/1 to 27/28 according to the yellow paper.
func (registrar *Registrar) RegisterCheckpoint(key *ecdsa.PrivateKey, index uint64, hash []byte, rnum *big.Int, rhash [32]byte, sigs [][]byte) (*types.Transaction, error) {
	var (
		r [][32]byte
		s [][32]byte
		v []uint8
	)
	for i := 0; i < len(sigs); i++ {
		if len(sigs[i]) != 65 {
			return nil, errors.New("invalid signature")
		}
		r = append(r, common.BytesToHash(sigs[i][:32]))
		s = append(s, common.BytesToHash(sigs[i][32:64]))
		v = append(v, sigs[i][64])
	}
	return registrar.contract.SetCheckpoint(bind.NewKeyedTransactor(key), rnum, rhash, common.BytesToHash(hash), index, v, r, s)
}
