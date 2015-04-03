package natspec

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/resolver"
	"github.com/ethereum/go-ethereum/core"
	//"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	xe "github.com/ethereum/go-ethereum/xeth"
)

type testFrontend struct {
	t        *testing.T
	ethereum *eth.Ethereum
	xeth     *xe.XEth
}

func (f *testFrontend) UnlockAccount(acc []byte) bool {
	f.t.Logf("Unlocking account %v\n", common.Bytes2Hex(acc))
	f.ethereum.AccountManager().Unlock(acc, "password")
	return true
}

func (testFrontend) ConfirmTransaction(message string) bool { return true }

var port = 30300

func testJEthRE(t *testing.T) (ethereum *eth.Ethereum, err error) {
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

func (self *testFrontend) insertTx(addr, contract, fnsig string, args []string) {

	hash := common.Bytes2Hex(crypto.Sha3([]byte(fnsig)))
	data := "0x" + hash[0:8]
	for _, arg := range args {
		data = data + common.Bytes2Hex(common.Hex2BytesFixed(arg, 32))
	}
	self.t.Logf("Tx data: %v", data)
	self.xeth.Transact(addr, contract, "100000000000", "100000", "100000", data)

	cb := common.HexToAddress(addr)
	stateDb := self.ethereum.ChainManager().State().Copy()

	coinbase := stateDb.GetStateObject(cb)
	coinbase.SetGasPool(big.NewInt(100000))
	block := self.ethereum.ChainManager().NewBlock(cb)
	txs := self.ethereum.TxPool().GetTransactions()
	tx := txs[0]

	_, gas, err := core.ApplyMessage(core.NewEnv(stateDb, self.ethereum.ChainManager(), tx, block), tx, coinbase)

	self.t.Logf("ApplyMessage: gas %v  err %v", gas, err)

	self.ethereum.TxPool().RemoveSet(txs)
	self.xeth = self.xeth.WithState(stateDb)

}

func TestNatspecE2E(t *testing.T) {
	ethereum, err := testJEthRE(t)
	if err != nil {
		t.Errorf("error creating jsre, got %v", err)
		return
	}
	err = ethereum.Start()
	if err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()

	frontend := &testFrontend{t: t, ethereum: ethereum}
	frontend.xeth = xe.New(ethereum, frontend)

	addr := frontend.xeth.Coinbase()
	if addr != "0x"+core.TestAccount {
		t.Errorf("CoinBase %v does not match TestAccount 0x%v", addr, core.TestAccount)
	}
	t.Logf("CoinBase is %v", addr)

	balance := frontend.xeth.BalanceAt(core.TestAccount)
	if balance != core.TestBalance {
		t.Errorf("Balance %v does not match TestBalance %v", balance, core.TestBalance)
	}
	t.Logf("Balance is %v", balance)

	frontend.insertTx(addr, core.ContractAddrURLhint, "register(bytes32,bytes32)", []string{"1234", "5678"})

	t.Logf("testcnt: %v", frontend.xeth.StorageAt(core.ContractAddrURLhint, "00"))

	for i := 0; i < 10; i++ {
		t.Logf("storage[%v] = %v", i, frontend.xeth.StorageAt("0x"+core.ContractAddrURLhint, fmt.Sprintf("%v", i)))
	}

	rsv := resolver.New(frontend.xeth, resolver.URLHintContractAddress, resolver.NameRegContractAddress)
	url, err2 := rsv.ContentHashToUrl(common.BytesToHash(common.Hex2BytesFixed("1234", 32)))

	t.Logf("URL: %v  err: %v", url, err2)

	/*
	   This test is unfinished; first we need to see the result of a
	   transaction in the contract storage (testcnt should be 1).
	*/

}
