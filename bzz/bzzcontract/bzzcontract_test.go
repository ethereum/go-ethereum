package bzzcontract

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	//"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc"
	xe "github.com/ethereum/go-ethereum/xeth"
)

type testFrontend struct {
	t        *testing.T
	ethereum *eth.Ethereum
	xeth     *xe.XEth
	api      *rpc.EthereumApi
	coinbase string
}

func (f *testFrontend) UnlockAccount(acc []byte) bool {
	f.t.Logf("Unlocking account %v\n", common.Bytes2Hex(acc))
	f.ethereum.AccountManager().Unlock(acc, "password")
	return true
}

func (f *testFrontend) ConfirmTransaction(tx *types.Transaction) bool {
	return true
}

var port = 30300

func testEth(t *testing.T) (ethereum *eth.Ethereum, err error) {
	os.RemoveAll("/tmp/eth/")
	err = os.MkdirAll("/tmp/eth/keys/e273f01c99144c438695e10f24926dc1f9fbf62d/", os.ModePerm)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	err = os.MkdirAll("/tmp/eth/data", os.ModePerm)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	ks := crypto.NewKeyStorePlain("/tmp/eth/keys")
	ioutil.WriteFile("/tmp/eth/keys/e273f01c99144c438695e10f24926dc1f9fbf62d/e273f01c99144c438695e10f24926dc1f9fbf62d",
		[]byte(`{"Id":"RhRXD+fNRKS4jx+7ZfEsNA==","Address":"4nPwHJkUTEOGleEPJJJtwfn79i0=","PrivateKey":"h4ACVpe74uIvi5Cg/2tX/Yrm2xdr3J7QoMbMtNX2CNc="}`), os.ModePerm)

	port++
	ethereum, err = eth.New(&eth.Config{
		DataDir:        "/tmp/eth",
		AccountManager: accounts.NewManager(ks),
		Port:           fmt.Sprintf("%d", port),
		MaxPeers:       10,
		Name:           "test",
	})

	if err != nil {
		t.Errorf("%v", err)
		return
	}

	return
}

func testInit(t *testing.T) (self *testFrontend) {

	ethereum, err := testEth(t)
	if err != nil {
		t.Errorf("error creating jsre, got %v", err)
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
	if addr != "0x"+core.TestAccount {
		t.Errorf("CoinBase %v does not match TestAccount 0x%v", addr, core.TestAccount)
	}
	t.Logf("CoinBase is %v", addr)

	balance := self.xeth.BalanceAt(core.TestAccount)
	t.Logf("Balance is %v", balance)

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
      "to": "0x` + contract + `",
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
	stateDb := self.ethereum.ChainManager().State().Copy()
	block := self.ethereum.ChainManager().NewBlock(cb)
	coinbase := stateDb.GetStateObject(cb)
	coinbase.SetGasPool(big.NewInt(1000000))
	txs := self.ethereum.TxPool().GetTransactions()

	for i := 0; i < len(txs); i++ {
		for _, tx := range txs {
			if tx.Nonce() == uint64(i) {
				_, gas, err := core.ApplyMessage(core.NewEnv(stateDb, self.ethereum.ChainManager(), tx, block), tx, coinbase)
				//self.ethereum.TxPool().RemoveSet([]*types.Transaction{tx})
				self.t.Logf("ApplyMessage: gas %v  err %v", gas, err)
			}
		}
	}

	self.ethereum.TxPool().RemoveSet(txs)
	self.xeth = self.xeth.WithState(stateDb)

}

func storageAddress(varidx uint32, key []byte) string {
	data := make([]byte, 64)
	binary.BigEndian.PutUint32(data[60:64], varidx)
	copy(data[0:32], key[0:32])
	return "0x" + common.Bytes2Hex(crypto.Sha3(data))
}

func TestSwarmContract(t *testing.T) {

	tf := testInit(t)
	defer tf.ethereum.Stop()

	tf.insertTx(tf.coinbase, core.ContractAddrSwarm, "signup(uint256)", []string{"1000"})
	tf.applyTxs()

	addr := common.Hex2BytesFixed(tf.coinbase[2:], 32)
	key := storageAddress(0, addr)
	data := tf.xeth.StorageAt("0x"+core.ContractAddrSwarm, key)
	key = key[:65] + "6"
	data2 := tf.xeth.StorageAt("0x"+core.ContractAddrSwarm, key)

	t.Logf("addr = %x  key = %v  data = %v, %v", addr, key, data, data2)

}
