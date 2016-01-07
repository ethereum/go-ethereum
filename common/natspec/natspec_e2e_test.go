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

package natspec

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/httpclient"
	"github.com/ethereum/go-ethereum/common/registrar"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/node"
	xe "github.com/ethereum/go-ethereum/xeth"
)

const (
	testAddress = "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	testBalance = "10000000000000000000"
	testKey     = "e6fab74a43941f82d89cb7faa408e227cdad3153c4720e540e855c19b15e6674"

	testFileName = "long_file_name_for_testing_registration_of_URLs_longer_than_32_bytes.content"

	testNotice = "Register key `utils.toHex(_key)` <- content `utils.toHex(_content)`"

	testExpNotice = "Register key 0xadd1a7d961cff0242089674ec2ef6fca671ab15e1fe80e38859fc815b98d88ab <- content 0xb3a2dea218de5d8bbe6c4645aadbf67b5ab00ecb1a9ec95dbdad6a0eed3e41a7"

	testExpNotice2 = `About to submit transaction (NatSpec notice error: abi key does not match any method): {"params":[{"to":"%s","data": "0x31e12c20"}]}`

	testExpNotice3 = `About to submit transaction (no NatSpec info found for contract: HashToHash: content hash not found for '0x1392c62d05b2d149e22a339c531157ae06b44d39a674cce500064b12b9aeb019'): {"params":[{"to":"%s","data": "0x300a3bbfb3a2dea218de5d8bbe6c4645aadbf67b5ab00ecb1a9ec95dbdad6a0eed3e41a7000000000000000000000000000000000000000000000000000000000000000000000000000000000000000066696c653a2f2f2f746573742e636f6e74656e74"}]}`
)

const (
	testUserDoc = `
{
  "methods": {
    "register(uint256,uint256)": {
      "notice":  "` + testNotice + `"
    }
  },
  "invariants": [
    { "notice": "" }
  ],
  "construction": [
    { "notice": "" }
  ]
}
`
	testAbiDefinition = `
[{
  "name": "register",
  "constant": false,
  "type": "function",
  "inputs": [{
    "name": "_key",
    "type": "uint256"
  }, {
    "name": "_content",
    "type": "uint256"
  }],
  "outputs": []
}]
`

	testContractInfo = `
{
	"userDoc": ` + testUserDoc + `,
	"abiDefinition": ` + testAbiDefinition + `
}
`
)

type testFrontend struct {
	t           *testing.T
	ethereum    *eth.Ethereum
	xeth        *xe.XEth
	wait        chan *big.Int
	lastConfirm string
	wantNatSpec bool
}

func (self *testFrontend) AskPassword() (string, bool) {
	return "", true
}

func (self *testFrontend) UnlockAccount(acc []byte) bool {
	self.ethereum.AccountManager().Unlock(common.BytesToAddress(acc), "password")
	return true
}

func (self *testFrontend) ConfirmTransaction(tx string) bool {
	if self.wantNatSpec {
		client := httpclient.New("/tmp/")
		self.lastConfirm = GetNotice(self.xeth, tx, client)
	}
	return true
}

func testEth(t *testing.T) (ethereum *eth.Ethereum, err error) {

	tmp, err := ioutil.TempDir("", "natspec-test")
	if err != nil {
		t.Fatal(err)
	}
	db, _ := ethdb.NewMemDatabase()
	addr := common.HexToAddress(testAddress)
	core.WriteGenesisBlockForTesting(db, core.GenesisAccount{addr, common.String2Big(testBalance)})
	ks := crypto.NewKeyStorePassphrase(filepath.Join(tmp, "keystore"), crypto.LightScryptN, crypto.LightScryptP)
	am := accounts.NewManager(ks)
	keyb, err := crypto.HexToECDSA(testKey)
	if err != nil {
		t.Fatal(err)
	}
	key := crypto.NewKeyFromECDSA(keyb)
	err = ks.StoreKey(key, "")
	if err != nil {
		t.Fatal(err)
	}

	err = am.Unlock(key.Address, "")
	if err != nil {
		t.Fatal(err)
	}

	// only use minimalistic stack with no networking
	return eth.New(&node.ServiceContext{EventMux: new(event.TypeMux)}, &eth.Config{
		AccountManager:          am,
		Etherbase:               common.HexToAddress(testAddress),
		PowTest:                 true,
		TestGenesisState:        db,
		GpoMinGasPrice:          common.Big1,
		GpobaseCorrectionFactor: 1,
		GpoMaxGasPrice:          common.Big1,
	})
}

func testInit(t *testing.T) (self *testFrontend) {
	// initialise and start minimal ethereum stack
	ethereum, err := testEth(t)
	if err != nil {
		t.Errorf("error creating ethereum: %v", err)
		return
	}
	err = ethereum.Start(nil)
	if err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}

	// mock frontend
	self = &testFrontend{t: t, ethereum: ethereum}
	self.xeth = xe.New(nil, self)
	self.wait = self.xeth.UpdateState()
	addr, _ := self.ethereum.Etherbase()

	// initialise the registry contracts
	reg := registrar.New(self.xeth)
	registrar.GlobalRegistrarAddr = "0x0"

	var txG, txH, txU string
	txG, err = reg.SetGlobalRegistrar("", addr)
	if err != nil {
		t.Fatalf("error creating GlobalRegistrar: %v", err)
	}
	if !processTxs(self, t, 1) {
		t.Fatalf("error mining txs")
	}
	recG := self.xeth.GetTxReceipt(common.HexToHash(txG))
	if recG == nil {
		t.Fatalf("blockchain error creating GlobalRegistrar")
	}
	registrar.GlobalRegistrarAddr = recG.ContractAddress.Hex()

	txH, err = reg.SetHashReg("", addr)
	if err != nil {
		t.Errorf("error creating HashReg: %v", err)
	}
	if !processTxs(self, t, 1) {
		t.Errorf("error mining txs")
	}
	recH := self.xeth.GetTxReceipt(common.HexToHash(txH))
	if recH == nil {
		t.Fatalf("blockchain error creating HashReg")
	}
	registrar.HashRegAddr = recH.ContractAddress.Hex()

	txU, err = reg.SetUrlHint("", addr)
	if err != nil {
		t.Errorf("error creating UrlHint: %v", err)
	}
	if !processTxs(self, t, 1) {
		t.Errorf("error mining txs")
	}
	recU := self.xeth.GetTxReceipt(common.HexToHash(txU))
	if recU == nil {
		t.Fatalf("blockchain error creating UrlHint")
	}
	registrar.UrlHintAddr = recU.ContractAddress.Hex()

	return
}

// end to end test
func TestNatspecE2E(t *testing.T) {
	t.Skip()

	tf := testInit(t)
	defer tf.ethereum.Stop()
	addr, _ := tf.ethereum.Etherbase()

	// create a contractInfo file (mock cloud-deployed contract metadocs)
	// incidentally this is the info for the HashReg contract itself
	ioutil.WriteFile("/tmp/"+testFileName, []byte(testContractInfo), os.ModePerm)
	dochash := crypto.Sha3Hash([]byte(testContractInfo))

	// take the codehash for the contract we wanna test
	codeb := tf.xeth.CodeAtBytes(registrar.HashRegAddr)
	codehash := crypto.Sha3Hash(codeb)

	reg := registrar.New(tf.xeth)
	_, err := reg.SetHashToHash(addr, codehash, dochash)
	if err != nil {
		t.Errorf("error registering: %v", err)
	}
	_, err = reg.SetUrlToHash(addr, dochash, "file:///"+testFileName)
	if err != nil {
		t.Errorf("error registering: %v", err)
	}
	if !processTxs(tf, t, 5) {
		return
	}

	// NatSpec info for register method of HashReg contract installed
	// now using the same transactions to check confirm messages

	tf.wantNatSpec = true // this is set so now the backend uses natspec confirmation
	_, err = reg.SetHashToHash(addr, codehash, dochash)
	if err != nil {
		t.Errorf("error calling contract registry: %v", err)
	}

	fmt.Printf("GlobalRegistrar: %v, HashReg: %v, UrlHint: %v\n", registrar.GlobalRegistrarAddr, registrar.HashRegAddr, registrar.UrlHintAddr)
	if tf.lastConfirm != testExpNotice {
		t.Errorf("Wrong confirm message. expected\n'%v', got\n'%v'", testExpNotice, tf.lastConfirm)
	}

	// test unknown method
	exp := fmt.Sprintf(testExpNotice2, registrar.HashRegAddr)
	_, err = reg.SetOwner(addr)
	if err != nil {
		t.Errorf("error setting owner: %v", err)
	}

	if tf.lastConfirm != exp {
		t.Errorf("Wrong confirm message, expected\n'%v', got\n'%v'", exp, tf.lastConfirm)
	}

	// test unknown contract
	exp = fmt.Sprintf(testExpNotice3, registrar.UrlHintAddr)

	_, err = reg.SetUrlToHash(addr, dochash, "file:///test.content")
	if err != nil {
		t.Errorf("error registering: %v", err)
	}

	if tf.lastConfirm != exp {
		t.Errorf("Wrong confirm message, expected '%v', got '%v'", exp, tf.lastConfirm)
	}

}

func pendingTransactions(repl *testFrontend, t *testing.T) (txc int64, err error) {
	txs := repl.ethereum.TxPool().GetTransactions()
	return int64(len(txs)), nil
}

func processTxs(repl *testFrontend, t *testing.T, expTxc int) bool {
	var txc int64
	var err error
	for i := 0; i < 50; i++ {
		txc, err = pendingTransactions(repl, t)
		if err != nil {
			t.Errorf("unexpected error checking pending transactions: %v", err)
			return false
		}
		if expTxc < int(txc) {
			t.Errorf("too many pending transactions: expected %v, got %v", expTxc, txc)
			return false
		} else if expTxc == int(txc) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if int(txc) != expTxc {
		t.Errorf("incorrect number of pending transactions, expected %v, got %v", expTxc, txc)
		return false
	}

	err = repl.ethereum.StartMining(runtime.NumCPU(), "")
	if err != nil {
		t.Errorf("unexpected error mining: %v", err)
		return false
	}
	defer repl.ethereum.StopMining()

	timer := time.NewTimer(100 * time.Second)
	height := new(big.Int).Add(repl.xeth.CurrentBlock().Number(), big.NewInt(1))
	repl.wait <- height
	select {
	case <-timer.C:
		// if times out make sure the xeth loop does not block
		go func() {
			select {
			case repl.wait <- nil:
			case <-repl.wait:
			}
		}()
	case <-repl.wait:
	}
	txc, err = pendingTransactions(repl, t)
	if err != nil {
		t.Errorf("unexpected error checking pending transactions: %v", err)
		return false
	}
	if txc != 0 {
		t.Errorf("%d trasactions were not mined", txc)
		return false
	}
	return true
}
