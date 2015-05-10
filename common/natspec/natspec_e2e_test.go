package natspec

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/docserver"
	"github.com/ethereum/go-ethereum/common/resolver"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	xe "github.com/ethereum/go-ethereum/xeth"
)

const (
	testBalance = "10000000000000000000"

	testFileName = "long_file_name_for_testing_registration_of_URLs_longer_than_32_bytes.content"

	testNotice = "Register key `utils.toHex(_key)` <- content `utils.toHex(_content)`"

	testExpNotice = "Register key 0xadd1a7d961cff0242089674ec2ef6fca671ab15e1fe80e38859fc815b98d88ab <- content 0xb3a2dea218de5d8bbe6c4645aadbf67b5ab00ecb1a9ec95dbdad6a0eed3e41a7"

	testExpNotice2 = `About to submit transaction (NatSpec notice error: abi key does not match any method): {"params":[{"to":"%s","data": "0x31e12c20"}]}`

	testExpNotice3 = `About to submit transaction (no NatSpec info found for contract: content hash not found for '0x1392c62d05b2d149e22a339c531157ae06b44d39a674cce500064b12b9aeb019'): {"params":[{"to":"%s","data": "0x300a3bbfb3a2dea218de5d8bbe6c4645aadbf67b5ab00ecb1a9ec95dbdad6a0eed3e41a7000000000000000000000000000000000000000000000000000000000000000000000000000000000000000066696c653a2f2f2f746573742e636f6e74656e74"}]}`
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
	t *testing.T
	// resolver    *resolver.Resolver
	ethereum    *eth.Ethereum
	xeth        *xe.XEth
	coinbase    common.Address
	stateDb     *state.StateDB
	txc         uint64
	lastConfirm string
	wantNatSpec bool
}

func (self *testFrontend) UnlockAccount(acc []byte) bool {
	self.ethereum.AccountManager().Unlock(acc, "password")
	return true
}

func (self *testFrontend) ConfirmTransaction(tx string) bool {
	if self.wantNatSpec {
		ds, err := docserver.New("/tmp/")
		if err != nil {
			self.t.Errorf("Error creating DocServer: %v", err)
		}
		self.lastConfirm = GetNotice(self.xeth, tx, ds)
	}
	return true
}

func testEth(t *testing.T) (ethereum *eth.Ethereum, err error) {

	os.RemoveAll("/tmp/eth-natspec/")

	err = os.MkdirAll("/tmp/eth-natspec/keys", os.ModePerm)
	if err != nil {
		panic(err)
	}

	// create a testAddress
	ks := crypto.NewKeyStorePassphrase("/tmp/eth-natspec/keys")
	am := accounts.NewManager(ks)
	testAccount, err := am.NewAccount("password")
	if err != nil {
		panic(err)
	}
	testAddress := common.Bytes2Hex(testAccount.Address)

	// set up mock genesis with balance on the testAddress
	core.GenesisData = []byte(`{
	"` + testAddress + `": {"balance": "` + testBalance + `"}
	}`)

	// only use minimalistic stack with no networking
	ethereum, err = eth.New(&eth.Config{
		DataDir:        "/tmp/eth-natspec",
		AccountManager: am,
		MaxPeers:       0,
	})

	if err != nil {
		panic(err)
	}

	return
}

func testInit(t *testing.T) (self *testFrontend) {
	// initialise and start minimal ethereum stack
	ethereum, err := testEth(t)
	if err != nil {
		t.Errorf("error creating ethereum: %v", err)
		return
	}
	err = ethereum.Start()
	if err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}

	// mock frontend
	self = &testFrontend{t: t, ethereum: ethereum}
	self.xeth = xe.New(ethereum, self)

	addr, _ := ethereum.Etherbase()
	self.coinbase = addr
	self.stateDb = self.ethereum.ChainManager().State().Copy()

	// initialise the registry contracts
	// self.resolver.CreateContracts(addr)
	resolver.New(self.xeth).CreateContracts(addr)
	self.applyTxs()
	// t.Logf("HashReg contract registered at %v", resolver.HashRegContractAddress)
	// t.Logf("URLHint contract registered at %v", resolver.UrlHintContractAddress)

	return

}

// this is needed for transaction to be applied to the state in testing
// the heavy lifing is done in XEth.ApplyTestTxs
// this is fragile,
// and does process leaking since xeth loops cannot quit safely
// should be replaced by proper mining with testDAG for easy full integration tests
func (self *testFrontend) applyTxs() {
	self.txc, self.xeth = self.xeth.ApplyTestTxs(self.stateDb, self.coinbase, self.txc)
	return
}

// end to end test
func TestNatspecE2E(t *testing.T) {
	// t.Skip()

	tf := testInit(t)
	defer tf.ethereum.Stop()

	// create a contractInfo file (mock cloud-deployed contract metadocs)
	// incidentally this is the info for the registry contract itself
	ioutil.WriteFile("/tmp/"+testFileName, []byte(testContractInfo), os.ModePerm)
	dochash := common.BytesToHash(crypto.Sha3([]byte(testContractInfo)))

	// take the codehash for the contract we wanna test
	// codehex := tf.xeth.CodeAt(resolver.HashRegContractAddress)
	codeb := tf.xeth.CodeAtBytes(resolver.HashRegContractAddress)
	codehash := common.BytesToHash(crypto.Sha3(codeb))

	// use resolver to register codehash->dochash->url
	registry := resolver.New(tf.xeth)
	_, err := registry.Register(tf.coinbase, codehash, dochash, "file:///"+testFileName)
	if err != nil {
		t.Errorf("error registering: %v", err)
	}
	// apply txs to the state
	tf.applyTxs()

	// NatSpec info for register method of HashReg contract installed
	// now using the same transactions to check confirm messages

	tf.wantNatSpec = true // this is set so now the backend uses natspec confirmation
	_, err = registry.RegisterContentHash(tf.coinbase, codehash, dochash)
	if err != nil {
		t.Errorf("error calling contract registry: %v", err)
	}

	if tf.lastConfirm != testExpNotice {
		t.Errorf("Wrong confirm message. expected '%v', got '%v'", testExpNotice, tf.lastConfirm)
	}

	// test unknown method
	exp := fmt.Sprintf(testExpNotice2, resolver.HashRegContractAddress)
	_, err = registry.SetOwner(tf.coinbase)
	if err != nil {
		t.Errorf("error setting owner: %v", err)
	}

	if tf.lastConfirm != exp {
		t.Errorf("Wrong confirm message, expected '%v', got '%v'", exp, tf.lastConfirm)
	}

	// test unknown contract
	exp = fmt.Sprintf(testExpNotice3, resolver.UrlHintContractAddress)

	_, err = registry.RegisterUrl(tf.coinbase, dochash, "file:///test.content")
	if err != nil {
		t.Errorf("error registering: %v", err)
	}

	if tf.lastConfirm != exp {
		t.Errorf("Wrong confirm message, expected '%v', got '%v'", exp, tf.lastConfirm)
	}

}
