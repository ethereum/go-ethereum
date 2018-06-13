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

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/registrar/contract"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// registrar contract address for mainnet and testnet.
	RegistrarAddr = map[common.Hash]common.Address{
		// params.MainnetGenesisHash: common.HexToAddress(""),
		// params.TestnetGenesisHash: common.HexToAddress(""),
		params.RinkebyGenesisHash: common.HexToAddress("0xc72f57e41e2498ad3dab92f665b0f21e2c4f4b79"),
	}
)

var errEventNotFound = errors.New("contract event not found")

type Registrar struct {
	contract *contract.Contract
}

// NewRegistrar binds checkpoint contract and returns a registrar instance.
func NewRegistrar(contractAddr common.Address, backend bind.ContractBackend) (*Registrar, error) {
	contract, err := contract.NewContract(contractAddr, backend)
	if err != nil {
		return nil, err
	}

	return &Registrar{
		contract: contract,
	}, nil
}

// WatchNewCheckpointEvent watches new fired NewCheckpointEvent and delivers all matching events by result channel.
func (registrar *Registrar) WatchNewCheckpointEvent(sink chan<- *contract.ContractNewCheckpointEvent) (event.Subscription, error) {
	return registrar.contract.WatchNewCheckpointEvent(nil, sink, nil)
}

// FilterNewCheckpointEvent filters out NewCheckpointEvent for specific section number.
func (registrar *Registrar) FilterNewCheckpointEvent(head, section, sectionSize, processConfirm uint64) (*contract.ContractNewCheckpointEventIterator, error) {
	start := (section+1)*sectionSize + processConfirm
	if head < start {
		return nil, errEventNotFound
	}
	opt := &bind.FilterOpts{
		Start: start,
		End:   &head,
	}
	return registrar.contract.FilterNewCheckpointEvent(opt, []*big.Int{big.NewInt(int64(section))})
}
