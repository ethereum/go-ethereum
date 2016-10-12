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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger/glog"
)

var (
	ForceJit  bool
	EnableJit bool
)

func init() {
	glog.SetV(0)
	if os.Getenv("JITVM") == "true" {
		ForceJit = true
		EnableJit = true
	}
}

func checkLogs(tlog []Log, logs vm.Logs) error {

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
			genBloom := common.LeftPadBytes(types.LogsBloom(vm.Logs{logs[i]}).Bytes(), 256)

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
	state.SetNonce(addr, common.Big(account.Nonce).Uint64())
	state.SetBalance(addr, common.Big(account.Balance))
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

type RuleSet struct {
	HomesteadBlock *big.Int
	DAOForkBlock   *big.Int
	DAOForkSupport bool
}

func (r RuleSet) IsHomestead(n *big.Int) bool {
	return n.Cmp(r.HomesteadBlock) >= 0
}

func NewEVMEnvironment(vmTest bool, ruleSet RuleSet, state *state.StateDB, envValues map[string]string, exeValues map[string]string) *vm.Environment {
	origin := common.HexToAddress(exeValues["caller"])
	if len(exeValues["secretKey"]) > 0 {
		key, _ := hex.DecodeString(exeValues["secretKey"])
		origin = crypto.PubkeyToAddress(crypto.ToECDSA(key).PublicKey)
	}

	context := vm.Context{
		Origin:      origin,
		Coinbase:    common.HexToAddress(envValues["currentCoinbase"]),
		BlockNumber: common.Big(envValues["currentNumber"]),
		Time:        common.Big(envValues["currentTimestamp"]),
		Difficulty:  common.Big(envValues["currentDifficulty"]),
		GasLimit:    common.Big(envValues["currentGasLimit"]),
		GasPrice:    common.Big(exeValues["gasPrice"]),
	}

	//var initialCall bool
	backend := &core.EVMBackend{
		GetHashFn: func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(big.NewInt(int64(n)).String())))
		},
		State: state,
	}
	initialCall := true
	canTransfer := func(db vm.Database, address common.Address, amount *big.Int) bool {
		if vmTest {
			if initialCall {
				initialCall = false
				return true
			}
		}
		return db.GetBalance(address).Cmp(amount) >= 0
	}
	transfer := func(db vm.Database, sender, recipient common.Address, amount *big.Int) {
		if vmTest {
			return
		}
		core.Transfer(db, sender, recipient, amount)
	}
	context.CallContext = core.EVMCallContext{canTransfer, transfer}

	env := vm.NewEnvironment(context, backend, ruleSet, vm.Config{Test: vmTest})
	return env
}

type Message struct {
	from              common.Address
	to                *common.Address
	value, gas, price *big.Int
	data              []byte
	nonce             uint64
}

func NewMessage(from common.Address, to *common.Address, data []byte, value, gas, price *big.Int, nonce uint64) Message {
	return Message{from, to, value, gas, price, data, nonce}
}

func (self Message) Hash() []byte                          { return nil }
func (self Message) From() (common.Address, error)         { return self.from, nil }
func (self Message) FromFrontier() (common.Address, error) { return self.from, nil }
func (self Message) To() *common.Address                   { return self.to }
func (self Message) GasPrice() *big.Int                    { return self.price }
func (self Message) Gas() *big.Int                         { return self.gas }
func (self Message) Value() *big.Int                       { return self.value }
func (self Message) Nonce() uint64                         { return self.nonce }
func (self Message) CheckNonce() bool                      { return true }
func (self Message) Data() []byte                          { return self.data }
