// Copyright 2014 The go-ethereum Authors
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

// Package eth implements the Ethereum protocol.
package eth

// WARNING: THIS CODE IS WORK IN PROGRESS AND NEEDS EXTENSIVE REVIEW
//
// Utility functions for preparing and enacting a hard fork to revert the hack of "The DAO"
// according to proposed spec at:
// https://docs.google.com/document/d/1VfuAH7Zf0UQmuVw1o7cTPbNtp1wFYzlN1tBKV6SbVSI
//

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/dao"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

var (
	theDAOAddr = common.HexToAddress("bb9bc244d798123fde783fcc1c72d3bb8c189413")
	// TODO: set these after community consensus
	theDAOReplacementCode = []byte{0x00}
	hardForkBlock         = big.NewInt(999999999)
)

type DAOHardForkData struct {
	ChildDAOs             []common.Address
	ExtraBalances         []common.Address
	TheDAOReplacementCode []byte
}

func TestRunHardFork(eth *Ethereum) error {
	hfData, err := prepareHardForkData(eth)
	if err != nil {
		return err
	}

	// log JSON representation of hard fork data for manual inspection
	logHardForkData(hfData, eth)

	// uncomment to (for testing) apply hard fork on current state:
	// err = applyHardFork(hfData, eth)
	return err
}

func logHardForkData(hfData *DAOHardForkData, eth *Ethereum) {
	j, _ := json.Marshal(hfData)
	glog.V(logger.Info).Infof("Generated hard fork data: %s", j)

	statedb, err := eth.BlockChain().State()
	if err != nil {
		return
	}

	// only accumulated for logging purposes
	total := big.NewInt(0)
	for i := 0; i < len(hfData.ChildDAOs); i++ {
		acc := statedb.GetAccount(hfData.ChildDAOs[i])
		total.Add(total, acc.Balance())
	}

	for i := 0; i < len(hfData.ExtraBalances); i++ {
		acc := statedb.GetAccount(hfData.ExtraBalances[i])
		total.Add(total, acc.Balance())
	}

	glog.V(logger.Info).Infof("Total wei in all accounts: %v", total)
}

func prepareHardForkData(eth *Ethereum) (*DAOHardForkData, error) {
	hfData := &DAOHardForkData{
		TheDAOReplacementCode: theDAOReplacementCode,
	}

	theDAO, err := dao.NewDAO(theDAOAddr, NewContractBackend(eth))
	if err != nil {
		glog.V(logger.Info).Infof("could not instantiate DAO: %v", err)
		return nil, err
	}

	// create list of "The DAO" and all its children, recursively
	daos, err := traverseDAOs(theDAO, eth)
	if err != nil {
		return nil, err
	}
	glog.V(logger.Info).Infof("DAOs traversed: %v", len(daos))

	// and their extraBalance accounts
	for i := 0; i < len(daos); i++ {
		bal, err := daos[i].ExtraBalance(nil)
		if err != nil {
			glog.V(logger.Info).Infof("could not get extraBalance for DAO: %v %v", i, err)
			return nil, err
		}
		hfData.ExtraBalances = append(hfData.ExtraBalances, bal)
	}

	for i := 0; i < len(daos); i++ {
		hfData.ChildDAOs = append(hfData.ChildDAOs, daos[i].Address())
	}

	return hfData, nil
}

func traverseDAOs(d *dao.DAO, eth *Ethereum) ([]*dao.DAO, error) {
	numberOfProposalsBig, err := d.NumberOfProposals(nil)
	if err != nil {
		return nil, err
	}
	numberOfProposals := int(numberOfProposalsBig.Uint64())

	var daos []*dao.DAO
	for i := 0; i < numberOfProposals; i++ {
		pId := big.NewInt(int64(i))
		proposal, err := d.Proposals(nil, pId)
		if err != nil {
			return nil, err
		}
		if proposal.NewCurator && proposal.ProposalPassed {
			childAddr, err := d.GetNewDAOAddress(nil, pId)
			if err != nil {
				glog.V(logger.Info).Infof("could not get child DAO address for proposalId: %v %v", pId, err)
				return nil, err
			}
			childDAO, err := dao.NewDAO(childAddr, NewContractBackend(eth))
			if err != nil {
				glog.V(logger.Info).Infof("could not instantiate childDAO: %x %v", childAddr, err)
				return nil, err
			}
			// append this child, then all its children (recursively)
			daos = append(daos, childDAO)
			children, err := traverseDAOs(childDAO, eth)
			if err != nil {
				glog.V(logger.Info).Infof("could not traverse children: %x %v", childAddr, err)
				return nil, err
			}
			daos = append(daos, children...)
		}
	}
	return daos, nil
}

// WARNING: THIS CODE DIRECTLY MODIFIES BLOCKCHAIN STATE, BYPASSING REGULAR
//          PROTOCOL RULES. REVIEW CAREFULLY!
// TODO: add tests!
func applyHardFork(hfData *DAOHardForkData, eth *Ethereum) error {
	statedb, err := eth.BlockChain().State()
	if err != nil {
		return err
	}

	// move all ether from all the children of the "The DAO"
	// as well as their extraBalance accounts back to "The DAO" account
	theDAOAcc := statedb.GetAccount(theDAOAddr)
	bigZero := big.NewInt(0)

	moveEther := func(addr common.Address) {
		acc := statedb.GetAccount(addr)
		bal := new(big.Int).Set(acc.Balance())
		acc.SetBalance(bigZero)
		theDAOAcc.AddBalance(bal)
	}

	for i := 0; i < len(hfData.ChildDAOs); i++ {
		moveEther(hfData.ChildDAOs[i])
	}
	for i := 0; i < len(hfData.ExtraBalances); i++ {
		moveEther(hfData.ExtraBalances[i])
	}

	// set new code of "The DAO" account (this also sets codeHash, see core/state/state_object.go)
	theDAOAcc.SetCode(hfData.TheDAOReplacementCode)

	// TODO: at this point, the account state (variables) have not changed;
	//       verify if they should be modified or can be kept unmodified.

	glog.V(logger.Info).Infof("wei in re-created DAO: %v", theDAOAcc.Balance())
	return nil
}
