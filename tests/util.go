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

package tests

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	ForceJit  bool
	EnableJit bool
)

func init() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlCrit, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
	if os.Getenv("JITVM") == "true" {
		ForceJit = true
		EnableJit = true
	}
}

func checkLogs(tlog []Log, logs []*types.Log) error {

	if len(tlog) != len(logs) {
		return fmt.Errorf("log length mismatch. Expected %d, got %d", len(tlog), len(logs))
	} else {
		for i, log := range tlog {
			if common.HexToAddress(log.AddressF) != logs[i].Address {
				return fmt.Errorf("log address expected %v got %x", log.AddressF, logs[i].Address)
			}

			if !bytes.Equal(logs[i].Data, common.FromHex(log.DataF)) {
				return fmt.Errorf("log data expected %v got %x", log.DataF, logs[i].Data)
			}

			if len(log.TopicsF) != len(logs[i].Topics) {
				return fmt.Errorf("log topics length expected %d got %d", len(log.TopicsF), logs[i].Topics)
			} else {
				for j, topic := range log.TopicsF {
					if common.HexToHash(topic) != logs[i].Topics[j] {
						return fmt.Errorf("log topic[%d] expected %v got %x", j, topic, logs[i].Topics[j])
					}
				}
			}
			genBloom := math.PaddedBigBytes(types.LogsBloom([]*types.Log{logs[i]}), 256)

			if !bytes.Equal(genBloom, common.Hex2Bytes(log.BloomF)) {
				return fmt.Errorf("bloom mismatch")
			}
		}
	}
	return nil
}

type Account struct {
	Balance string
	Code    string
	Nonce   string
	Storage map[string]string
}

type Log struct {
	AddressF string   `json:"address"`
	DataF    string   `json:"data"`
	TopicsF  []string `json:"topics"`
	BloomF   string   `json:"bloom"`
}

func (self Log) Address() []byte      { return common.Hex2Bytes(self.AddressF) }
func (self Log) Data() []byte         { return common.Hex2Bytes(self.DataF) }
func (self Log) RlpData() interface{} { return nil }
func (self Log) Topics() [][]byte {
	t := make([][]byte, len(self.TopicsF))
	for i, topic := range self.TopicsF {
		t[i] = common.Hex2Bytes(topic)
	}
	return t
}

func makePreState(db ethdb.Database, accounts map[string]Account) *state.StateDB {
	statedb, _ := state.New(common.Hash{}, db)
	for addr, account := range accounts {
		insertAccount(statedb, addr, account)
	}
	return statedb
}

func insertAccount(state *state.StateDB, saddr string, account Account) {
	if common.IsHex(account.Code) {
		account.Code = account.Code[2:]
	}
	addr := common.HexToAddress(saddr)
	state.SetCode(addr, common.Hex2Bytes(account.Code))
	state.SetNonce(addr, math.MustParseUint64(account.Nonce))
	state.SetBalance(addr, math.MustParseBig256(account.Balance))
	for a, v := range account.Storage {
		state.SetState(addr, common.HexToHash(a), common.HexToHash(v))
	}
}

type VmEnv struct {
	CurrentCoinbase   string
	CurrentDifficulty string
	CurrentGasLimit   string
	CurrentNumber     string
	CurrentTimestamp  interface{}
	PreviousHash      string
}

type VmTest struct {
	Callcreates interface{}
	//Env         map[string]string
	Env           VmEnv
	Exec          map[string]string
	Transaction   map[string]string
	Logs          []Log
	Gas           string
	Out           string
	Post          map[string]Account
	Pre           map[string]Account
	PostStateRoot string
}

func NewEVMEnvironment(vmTest bool, chainConfig *params.ChainConfig, statedb *state.StateDB, envValues map[string]string, tx map[string]string) (*vm.EVM, core.Message) {
	var (
		data  = common.FromHex(tx["data"])
		gas   = math.MustParseBig256(tx["gasLimit"])
		price = math.MustParseBig256(tx["gasPrice"])
		value = math.MustParseBig256(tx["value"])
		nonce = math.MustParseUint64(tx["nonce"])
	)

	origin := common.HexToAddress(tx["caller"])
	if len(tx["secretKey"]) > 0 {
		key, _ := hex.DecodeString(tx["secretKey"])
		origin = crypto.PubkeyToAddress(crypto.ToECDSA(key).PublicKey)
	}

	var to *common.Address
	if len(tx["to"]) > 2 {
		t := common.HexToAddress(tx["to"])
		to = &t
	}

	msg := types.NewMessage(origin, to, nonce, value, gas, price, data, true)

	initialCall := true
	canTransfer := func(db vm.StateDB, address common.Address, amount *big.Int) bool {
		if vmTest {
			if initialCall {
				initialCall = false
				return true
			}
		}
		return core.CanTransfer(db, address, amount)
	}
	transfer := func(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
		if vmTest {
			return
		}
		core.Transfer(db, sender, recipient, amount)
	}

	context := vm.Context{
		CanTransfer: canTransfer,
		Transfer:    transfer,
		GetHash: func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(big.NewInt(int64(n)).String())))
		},

		Origin:      origin,
		Coinbase:    common.HexToAddress(envValues["currentCoinbase"]),
		BlockNumber: math.MustParseBig256(envValues["currentNumber"]),
		Time:        math.MustParseBig256(envValues["currentTimestamp"]),
		GasLimit:    math.MustParseBig256(envValues["currentGasLimit"]),
		Difficulty:  math.MustParseBig256(envValues["currentDifficulty"]),
		GasPrice:    price,
	}
	if context.GasPrice == nil {
		context.GasPrice = new(big.Int)
	}
	return vm.NewEVM(context, statedb, chainConfig, vm.Config{NoRecursion: vmTest}), msg
}
