package natspec

import (
	"io/ioutil"
	"math/big"
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
	"github.com/ethereum/go-ethereum/rpc"
	xe "github.com/ethereum/go-ethereum/xeth"
)

type testFrontend struct {
	t           *testing.T
	ethereum    *eth.Ethereum
	xeth        *xe.XEth
	api         *rpc.EthereumApi
	coinbase    string
	stateDb     *state.StateDB
	txc         uint64
	lastConfirm string
	makeNatSpec bool
}

const (
	testAccount = "e273f01c99144c438695e10f24926dc1f9fbf62d"
	testBalance = "1000000000000"
)

const testFileName = "long_file_name_for_testing_registration_of_URLs_longer_than_32_bytes.content"

const testNotice = "Register key `utils.toHex(_key)` <- content `utils.toHex(_content)`"
const testExpNotice = "Register key 0xadd1a7d961cff0242089674ec2ef6fca671ab15e1fe80e38859fc815b98d88ab <- content 0xc00d5bcc872e17813df6ec5c646bb281a6e2d3b454c2c400c78192adf3344af9"
const testExpNotice2 = `About to submit transaction (NatSpec notice error "abi key does not match any method"): {"id":6,"jsonrpc":"2.0","method":"eth_transact","params":[{"from":"0xe273f01c99144c438695e10f24926dc1f9fbf62d","to":"0xb737b91f8e95cf756766fc7c62c9a8ff58470381","value":"100000000000","gas":"100000","gasPrice":"100000","data":"0x31e12c20"}]}`
const testExpNotice3 = `About to submit transaction (no NatSpec info found for contract): {"id":6,"jsonrpc":"2.0","method":"eth_transact","params":[{"from":"0xe273f01c99144c438695e10f24926dc1f9fbf62d","to":"0x8b839ad85686967a4f418eccc81962eaee314ac3","value":"100000000000","gas":"100000","gasPrice":"100000","data":"0x300a3bbfc00d5bcc872e17813df6ec5c646bb281a6e2d3b454c2c400c78192adf3344af900000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000"}]}`

const testUserDoc = `
{
  "source": "...",
  "language": "Solidity",
  "languageVersion": 1,
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

const testABI = `
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

const testDocs = `
{
	"userdoc": ` + testUserDoc + `,
	"abi": ` + testABI + `
}
`

func (f *testFrontend) UnlockAccount(acc []byte) bool {
	f.t.Logf("Unlocking account %v\n", common.Bytes2Hex(acc))
	f.ethereum.AccountManager().Unlock(acc, "password")
	return true
}

func (f *testFrontend) ConfirmTransaction(tx string) bool {
	//f.t.Logf("ConfirmTransaction called  tx = %v", tx)
	if f.makeNatSpec {
		ds, err := docserver.New("/tmp/")
		if err != nil {
			f.t.Errorf("Error creating DocServer: %v", err)
		}
		f.lastConfirm = GetNotice(f.xeth, tx, ds)
	}
	return true
}

var port = 30300

func testEth(t *testing.T) (ethereum *eth.Ethereum, err error) {
	os.RemoveAll("/tmp/eth-natspec/")
	err = os.MkdirAll("/tmp/eth-natspec/keys/e273f01c99144c438695e10f24926dc1f9fbf62d/", os.ModePerm)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	err = os.MkdirAll("/tmp/eth-natspec/data", os.ModePerm)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	ks := crypto.NewKeyStorePlain("/tmp/eth-natspec/keys")
	ioutil.WriteFile("/tmp/eth-natspec/keys/e273f01c99144c438695e10f24926dc1f9fbf62d/e273f01c99144c438695e10f24926dc1f9fbf62d",
		[]byte(`{"Id":"RhRXD+fNRKS4jx+7ZfEsNA==","Address":"4nPwHJkUTEOGleEPJJJtwfn79i0=","PrivateKey":"h4ACVpe74uIvi5Cg/2tX/Yrm2xdr3J7QoMbMtNX2CNc="}`), os.ModePerm)

	port++
	ethereum, err = eth.New(&eth.Config{
		DataDir:        "/tmp/eth-natspec",
		AccountManager: accounts.NewManager(ks),
		Name:           "test",
	})

	if err != nil {
		t.Errorf("%v", err)
		return
	}

	return
}

func testInit(t *testing.T) (self *testFrontend) {

	core.GenesisData = []byte(`{
	"` + testAccount + `": {"balance": "` + testBalance + `"}
	}`)

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

	self = &testFrontend{t: t, ethereum: ethereum}
	self.xeth = xe.New(ethereum, self)
	self.api = rpc.NewEthereumApi(self.xeth)

	addr := self.xeth.Coinbase()
	self.coinbase = addr
	if addr != "0x"+testAccount {
		t.Errorf("CoinBase %v does not match TestAccount 0x%v", addr, testAccount)
	}
	t.Logf("CoinBase is %v", addr)

	balance := self.xeth.BalanceAt(testAccount)
	/*if balance != core.TestBalance {
		t.Errorf("Balance %v does not match TestBalance %v", balance, core.TestBalance)
	}*/
	t.Logf("Balance is %v", balance)

	self.stateDb = self.ethereum.ChainManager().State().Copy()

	return

}

func (self *testFrontend) insertTx(addr, contract, fnsig string, args []string) {

	//cb := common.HexToAddress(self.coinbase)
	//coinbase := self.ethereum.ChainManager().State().GetStateObject(cb)

	hash := common.Bytes2Hex(crypto.Sha3([]byte(fnsig)))
	data := "0x" + hash[0:8]
	for _, arg := range args {
		data = data + common.Bytes2Hex(common.Hex2BytesFixed(arg, 32))
	}
	self.t.Logf("Tx data: %v", data)

	jsontx := `
[{
	  "from": "` + addr + `",
      "to": "` + contract + `",
	  "value": "100000000000",
	  "gas": "100000",
	  "gasPrice": "100000",
      "data": "` + data + `"
}]
`
	req := &rpc.RpcRequest{
		Jsonrpc: "2.0",
		Method:  "eth_transact",
		Params:  []byte(jsontx),
		Id:      6,
	}

	var reply interface{}
	err0 := self.api.GetRequestReply(req, &reply)
	if err0 != nil {
		self.t.Errorf("GetRequestReply error: %v", err0)
	}

	//self.xeth.Transact(addr, contract, "100000000000", "100000", "100000", data)

}

func (self *testFrontend) applyTxs() {

	cb := common.HexToAddress(self.coinbase)
	block := self.ethereum.ChainManager().NewBlock(cb)
	coinbase := self.stateDb.GetStateObject(cb)
	coinbase.SetGasPool(big.NewInt(10000000))
	txs := self.ethereum.TxPool().GetQueuedTransactions()

	for i := 0; i < len(txs); i++ {
		for _, tx := range txs {
			//self.t.Logf("%v %v %v", i, tx.Nonce(), self.txc)
			if tx.Nonce() == self.txc {
				_, gas, err := core.ApplyMessage(core.NewEnv(self.stateDb, self.ethereum.ChainManager(), tx, block), tx, coinbase)
				//self.ethereum.TxPool().RemoveSet([]*types.Transaction{tx})
				self.t.Logf("ApplyMessage: gas %v  err %v", gas, err)
				self.txc++
			}
		}
	}

	//self.ethereum.TxPool().RemoveSet(txs)
	self.xeth = self.xeth.WithState(self.stateDb)

}

func (self *testFrontend) registerURL(hash common.Hash, url string) {
	hashHex := common.Bytes2Hex(hash[:])
	urlBytes := []byte(url)
	var bb bool = true
	var cnt byte
	for bb {
		bb = len(urlBytes) > 0
		urlb := urlBytes
		if len(urlb) > 32 {
			urlb = urlb[:32]
		}
		urlHex := common.Bytes2Hex(urlb)
		self.insertTx(self.coinbase, resolver.URLHintContractAddress, "register(uint256,uint8,uint256)", []string{hashHex, common.Bytes2Hex([]byte{cnt}), urlHex})
		if len(urlBytes) > 32 {
			urlBytes = urlBytes[32:]
		} else {
			urlBytes = nil
		}
		cnt++
	}
}

func (self *testFrontend) setOwner() {

	self.insertTx(self.coinbase, resolver.HashRegContractAddress, "setowner()", []string{})

	/*owner := self.xeth.StorageAt("0x"+resolver.HashRegContractAddress, "0x0000000000000000000000000000000000000000000000000000000000000000")
	self.t.Logf("owner = %v", owner)
	if owner != self.coinbase {
		self.t.Errorf("setowner() unsuccessful, owner != coinbase")
	}*/
}

func (self *testFrontend) registerNatSpec(codehash, dochash common.Hash) {

	codeHex := common.Bytes2Hex(codehash[:])
	docHex := common.Bytes2Hex(dochash[:])
	self.insertTx(self.coinbase, resolver.HashRegContractAddress, "register(uint256,uint256)", []string{codeHex, docHex})
}

func (self *testFrontend) testResolver() *resolver.Resolver {
	return resolver.New(self.xeth, resolver.URLHintContractAddress, resolver.HashRegContractAddress)
}

func TestNatspecE2E(t *testing.T) {
	t.Skip()

	tf := testInit(t)
	defer tf.ethereum.Stop()

	resolver.CreateContracts(tf.xeth, testAccount)
	t.Logf("URLHint contract registered at %v", resolver.URLHintContractAddress)
	t.Logf("HashReg contract registered at %v", resolver.HashRegContractAddress)
	tf.applyTxs()

	ioutil.WriteFile("/tmp/"+testFileName, []byte(testDocs), os.ModePerm)
	dochash := common.BytesToHash(crypto.Sha3([]byte(testDocs)))

	codehex := tf.xeth.CodeAt(resolver.HashRegContractAddress)
	codehash := common.BytesToHash(crypto.Sha3(common.Hex2Bytes(codehex[2:])))

	tf.setOwner()
	tf.registerNatSpec(codehash, dochash)
	tf.registerURL(dochash, "file:///"+testFileName)
	tf.applyTxs()

	chash, err := tf.testResolver().KeyToContentHash(codehash)
	if err != nil {
		t.Errorf("Can't find content hash")
	}
	t.Logf("chash = %x  err = %v", chash, err)
	url, err2 := tf.testResolver().ContentHashToUrl(dochash)
	if err2 != nil {
		t.Errorf("Can't find URL hint")
	}
	t.Logf("url = %v  err = %v", url, err2)

	// NatSpec info for register method of HashReg contract installed
	// now using the same transactions to check confirm messages

	tf.makeNatSpec = true
	tf.registerNatSpec(codehash, dochash)
	t.Logf("Confirm message: %v\n", tf.lastConfirm)
	if tf.lastConfirm != testExpNotice {
		t.Errorf("Wrong confirm message, expected '%v', got '%v'", testExpNotice, tf.lastConfirm)
	}

	tf.setOwner()
	t.Logf("Confirm message for unknown method: %v\n", tf.lastConfirm)
	if tf.lastConfirm != testExpNotice2 {
		t.Errorf("Wrong confirm message, expected '%v', got '%v'", testExpNotice2, tf.lastConfirm)
	}

	tf.registerURL(dochash, "file:///test.content")
	t.Logf("Confirm message for unknown contract: %v\n", tf.lastConfirm)
	if tf.lastConfirm != testExpNotice3 {
		t.Errorf("Wrong confirm message, expected '%v', got '%v'", testExpNotice3, tf.lastConfirm)
	}

}
