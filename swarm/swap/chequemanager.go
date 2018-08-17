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

package swap

import (
	"math/big"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/swarm/state"
)

type ChequeManager struct {
	stateStore        state.Store
	serialPerNode     map[discover.NodeID]uint64
	openDebitCheques  map[discover.NodeID][]*Cheque
	openCreditCheques map[discover.NodeID][]*Cheque
}

type Cheque struct {
	serial       uint64
	timeout      uint64
	amount       *big.Int
	sumCumulated *big.Int
	beneficiary  discover.NodeID //this should probably be common.Address?
}

func NewChequeManager(stateStore state.Store) *ChequeManager {
	return &ChequeManager{
		stateStore: stateStore,
		//TODO: restore from state store
		serialPerNode:     make(map[discover.NodeID]uint64),
		openDebitCheques:  make(map[discover.NodeID][]*Cheque),
		openCreditCheques: make(map[discover.NodeID][]*Cheque),
	}
}

func (mgr *ChequeManager) CreateCheque(beneficiary discover.NodeID, amount *big.Int) *Cheque {
	mgr.serialPerNode[beneficiary]++
	cheque := &Cheque{
		serial:      mgr.serialPerNode[beneficiary],
		beneficiary: beneficiary,
		amount:      amount,
	}
	openCheques := mgr.openDebitCheques[beneficiary]
	if openCheques == nil {
		openCheques = make([]*Cheque, 0)
	}
	openCheques = append(openCheques, cheque)
	mgr.openDebitCheques[beneficiary] = openCheques
	return cheque
}
