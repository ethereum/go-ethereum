// Copyright 2015 The go-ethereum Authors
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

package versions

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	jsonlogger = logger.NewJsonLogger()
	// TODO: add Frontier address
	GlobalVersionsAddr   = common.HexToAddress("0x40bebcadbb4456db23fda39f261f3b2509096e9e") // test
	dummySender          = common.HexToAddress("0x16db48070243bc37a1c59cd5bb977ad7047618be") // test
	getVersionsSignature = "GetVersions()"
	firstCheckTime       = time.Second * 4
	continousCheckTime   = time.Second * 600
)

type VersionCheck struct {
	serverName string
	timer      *time.Timer
	e          *eth.Ethereum
	stop       chan bool
}

// Boilerplate to satisfy node.Service interface
func (v *VersionCheck) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

func (v *VersionCheck) APIs() []rpc.API {
	return []rpc.API{}
}

func (v *VersionCheck) Start(server *p2p.Server) error {
	v.serverName = server.Name
	// Check version first time after a few seconds so it shows after
	// other startup messages
	t := time.NewTimer(firstCheckTime)
	v.timer = t
	v.stop = make(chan bool)
	versionCheck := func() {
		for {
			select {
			case <-v.stop:
				close(v.stop)
				return
			case <-v.timer.C:
				_, err := get(v.e, v.serverName)
				if err != nil {
					glog.V(logger.Error).Infof("Could not query geth version contract: %s", err)
				}
				v.timer.Reset(continousCheckTime)
			}
		}
	}
	go versionCheck()
	return nil
}

func (v *VersionCheck) Stop() error {
	v.stop <- true
	select {
	case <-v.stop:
	}
	return nil
}

func NewVersionCheck(ctx *node.ServiceContext) (node.Service, error) {
	var v VersionCheck
	var e *eth.Ethereum
	// sets e to the Ethereum instance previously started
	// expects double pointer
	ctx.Service(&e)
	v.e = e
	return &v, nil
}

// query versions list from the (custom) accessor in the versions contract
func get(e *eth.Ethereum, clientVersion string) (string, error) {
	// TODO: move common/registrar abiSignature to some util package
	abi := crypto.Sha3([]byte(getVersionsSignature))[:4]
	res, _, err := simulateCall(
		e,
		&dummySender,
		&GlobalVersionsAddr,
		big.NewInt(3000000), // gasLimit
		big.NewInt(1),       // gasPrice
		big.NewInt(0),       // value
		abi)
	if err != nil {
		return "", err
	}

	// TODO: we use static arrays of size versionCount as workaround
	// until solidity has proper support for returning dynamic arrays
	versionCount := 10

	if len(res) != 2+(64*versionCount*3) { // 0x + three 32-byte fields per version
		return "", fmt.Errorf("unexpected result length from GetVersions")
	}

	// TODO: use ABI (after solidity supports returning arrays of arrays and/or structs)
	var versions []string
	var timestamps []uint64
	var signerCounts []uint64

	// trim 0x
	res = res[2:]

	// parse res
	for i := 0; i < versionCount; i++ {
		bytes := common.FromHex(res[:64])
		versions = append(versions, string(bytes))
		res = res[64:]
	}

	for i := 0; i < versionCount; i++ {
		ts, err := strconv.ParseUint(res[:64], 16, 64)
		if err != nil {
			return "", err
		}
		timestamps = append(timestamps, ts)
		res = res[64:]
	}

	for i := 0; i < versionCount; i++ {
		sc, err := strconv.ParseUint(res[:64], 16, 64)
		if err != nil {
			return "", err
		}
		signerCounts = append(signerCounts, sc)
		res = res[64:]
	}

	// TODO: version matching logic (e.g. most votes / most recent)
	if versions[0] != clientVersion {
		glog.V(logger.Info).Infof("geth version %s does not match recommended version %s", clientVersion, versions[0])
	}

	return res, nil
}

func simulateCall(e *eth.Ethereum, from0, to *common.Address, gas, gasPrice, value *big.Int, data []byte) (string, *big.Int, error) {
	stateCopy, err := e.BlockChain().State()
	if err != nil {
		return "", nil, err
	}
	from := stateCopy.GetOrNewStateObject(*from0)
	from.SetBalance(common.MaxBig)

	msg := callmsg{
		from:     from,
		to:       to,
		gas:      gas,
		gasPrice: gasPrice,
		value:    value,
		data:     data,
	}

	// Execute the call and return
	vmenv := core.NewEnv(stateCopy, e.BlockChain(), msg, e.BlockChain().CurrentHeader())
	gp := new(core.GasPool).AddGas(common.MaxBig)

	res, gas, err := core.ApplyMessage(vmenv, msg, gp)
	return common.ToHex(res), gas, err

}

// TODO: consider moving to package common or accounts/abi as it's useful for anyone
// simulating EVM CALL
type callmsg struct {
	from          *state.StateObject
	to            *common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m callmsg) From() (common.Address, error) { return m.from.Address(), nil }
func (m callmsg) Nonce() uint64                 { return m.from.Nonce() }
func (m callmsg) To() *common.Address           { return m.to }
func (m callmsg) GasPrice() *big.Int            { return m.gasPrice }
func (m callmsg) Gas() *big.Int                 { return m.gas }
func (m callmsg) Value() *big.Int               { return m.value }
func (m callmsg) Data() []byte                  { return m.data }
