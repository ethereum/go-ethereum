package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/common/docserver"
	"github.com/ethereum/go-ethereum/common/natspec"
	"github.com/ethereum/go-ethereum/common/resolver"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
)

const (
	testSolcPath = ""

	testKey     = "e6fab74a43941f82d89cb7faa408e227cdad3153c4720e540e855c19b15e6674"
	testAddress = "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	testBalance = "10000000000000000000"
	// of empty string
	testHash = "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
)

var (
	testGenesis = `{"` + testAddress[2:] + `": {"balance": "` + testBalance + `"}}`
)

type testjethre struct {
	*jsre
	stateDb     *state.StateDB
	lastConfirm string
	ds          *docserver.DocServer
}

func (self *testjethre) UnlockAccount(acc []byte) bool {
	err := self.ethereum.AccountManager().Unlock(acc, "")
	if err != nil {
		panic("unable to unlock")
	}
	return true
}

func (self *testjethre) ConfirmTransaction(tx string) bool {
	if self.ethereum.NatSpec {
		self.lastConfirm = natspec.GetNotice(self.xeth, tx, self.ds)
	}
	return true
}

func testJEthRE(t *testing.T) (string, *testjethre, *eth.Ethereum) {
	tmp, err := ioutil.TempDir("", "geth-test")
	if err != nil {
		t.Fatal(err)
	}

	// set up mock genesis with balance on the testAddress
	core.GenesisData = []byte(testGenesis)

	ks := crypto.NewKeyStorePassphrase(filepath.Join(tmp, "keys"))
	am := accounts.NewManager(ks)
	ethereum, err := eth.New(&eth.Config{
		DataDir:        tmp,
		AccountManager: am,
		MaxPeers:       0,
		Name:           "test",
	})
	if err != nil {
		t.Fatal("%v", err)
	}

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

	assetPath := path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist", "assets", "ext")
	ds, err := docserver.New("/")
	if err != nil {
		t.Errorf("Error creating DocServer: %v", err)
	}
	tf := &testjethre{ds: ds, stateDb: ethereum.ChainManager().State().Copy()}
	repl := newJSRE(ethereum, assetPath, testSolcPath, "", false, tf)
	tf.jsre = repl
	return tmp, tf, ethereum
}

// this line below is needed for transaction to be applied to the state in testing
// the heavy lifing is done in XEth.ApplyTestTxs
// this is fragile, overwriting xeth will result in
// process leaking since xeth loops cannot quit safely
// should be replaced by proper mining with testDAG for easy full integration tests
// txc, self.xeth = self.xeth.ApplyTestTxs(self.xeth.repl.stateDb, coinbase, txc)

func TestNodeInfo(t *testing.T) {
	tmp, repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Fatalf("error starting ethereum: %v", err)
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)
	want := `{"DiscPort":0,"IP":"0.0.0.0","ListenAddr":"","Name":"test","NodeID":"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","NodeUrl":"enode://00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000@0.0.0.0:0","TCPPort":0,"Td":"0"}`
	checkEvalJSON(t, repl, `admin.nodeInfo()`, want)
}

func TestAccounts(t *testing.T) {
	tmp, repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Fatalf("error starting ethereum: %v", err)
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)

	checkEvalJSON(t, repl, `eth.accounts`, `["`+testAddress+`"]`)
	checkEvalJSON(t, repl, `eth.coinbase`, `"`+testAddress+`"`)

	val, err := repl.re.Run(`admin.newAccount("password")`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	addr := val.String()
	if !regexp.MustCompile(`0x[0-9a-f]{40}`).MatchString(addr) {
		t.Errorf("address not hex: %q", addr)
	}

	// skip until order fixed #824
	// checkEvalJSON(t, repl, `eth.accounts`, `["`+testAddress+`", "`+addr+`"]`)
	// checkEvalJSON(t, repl, `eth.coinbase`, `"`+testAddress+`"`)
}

func TestBlockChain(t *testing.T) {
	tmp, repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Fatalf("error starting ethereum: %v", err)
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)
	// get current block dump before export/import.
	val, err := repl.re.Run("JSON.stringify(admin.debug.dumpBlock())")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	beforeExport := val.String()

	// do the export
	extmp, err := ioutil.TempDir("", "geth-test-export")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(extmp)
	tmpfile := filepath.Join(extmp, "export.chain")
	tmpfileq := strconv.Quote(tmpfile)

	checkEvalJSON(t, repl, `admin.export(`+tmpfileq+`)`, `true`)
	if _, err := os.Stat(tmpfile); err != nil {
		t.Fatal(err)
	}

	// check import, verify that dumpBlock gives the same result.
	checkEvalJSON(t, repl, `admin.import(`+tmpfileq+`)`, `true`)
	checkEvalJSON(t, repl, `admin.debug.dumpBlock()`, beforeExport)
}

func TestMining(t *testing.T) {
	tmp, repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Fatalf("error starting ethereum: %v", err)
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)
	checkEvalJSON(t, repl, `eth.mining`, `false`)
}

func TestRPC(t *testing.T) {
	tmp, repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)

	checkEvalJSON(t, repl, `admin.startRPC("127.0.0.1", 5004)`, `true`)
}

func TestCheckTestAccountBalance(t *testing.T) {
	tmp, repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)

	repl.re.Run(`primary = "` + testAddress + `"`)
	checkEvalJSON(t, repl, `eth.getBalance(primary)`, `"`+testBalance+`"`)
}

func TestSignature(t *testing.T) {
	tmp, repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)

	val, err := repl.re.Run(`eth.sign({from: "` + testAddress + `", data: "` + testHash + `"})`)

	// This is a very preliminary test, lacking actual signature verification
	if err != nil {
		t.Errorf("Error runnig js: %v", err)
		return
	}
	output := val.String()
	t.Logf("Output: %v", output)

	regex := regexp.MustCompile(`^0x[0-9a-f]{130}$`)
	if !regex.MatchString(output) {
		t.Errorf("Signature is not 65 bytes represented in hexadecimal.")
		return
	}
}

func TestContract(t *testing.T) {

	tmp, repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)

	var txc uint64
	coinbase := common.HexToAddress(testAddress)
	resolver.New(repl.xeth).CreateContracts(coinbase)

	source := `contract test {\n` +
		"   /// @notice Will multiply `a` by 7." + `\n` +
		`   function multiply(uint a) returns(uint d) {\n` +
		`       return a * 7;\n` +
		`   }\n` +
		`}\n`

	checkEvalJSON(t, repl, `admin.contractInfo.stop()`, `true`)

	contractInfo, err := ioutil.ReadFile("info_test.json")
	if err != nil {
		t.Fatalf("%v", err)
	}
	checkEvalJSON(t, repl, `primary = eth.accounts[0]`, `"`+testAddress+`"`)
	checkEvalJSON(t, repl, `source = "`+source+`"`, `"`+source+`"`)

	_, err = compiler.New("")
	if err != nil {
		t.Logf("solc not found: skipping compiler test")
		info, err := ioutil.ReadFile("info_test.json")
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = repl.re.Run(`contract = JSON.parse(` + strconv.Quote(string(info)) + `)`)
		if err != nil {
			t.Errorf("%v", err)
		}
	} else {
		checkEvalJSON(t, repl, `contract = eth.compile.solidity(source)`, string(contractInfo))
	}
	checkEvalJSON(t, repl, `contract.code`, `"605280600c6000396000f3006000357c010000000000000000000000000000000000000000000000000000000090048063c6888fa114602e57005b60376004356041565b8060005260206000f35b6000600782029050604d565b91905056"`)

	checkEvalJSON(
		t, repl,
		`contractaddress = eth.sendTransaction({from: primary, data: contract.code })`,
		`"0x5dcaace5982778b409c524873b319667eba5d074"`,
	)

	callSetup := `abiDef = JSON.parse('[{"constant":false,"inputs":[{"name":"a","type":"uint256"}],"name":"multiply","outputs":[{"name":"d","type":"uint256"}],"type":"function"}]');
Multiply7 = eth.contract(abiDef);
multiply7 = new Multiply7(contractaddress);
`

	_, err = repl.re.Run(callSetup)
	if err != nil {
		t.Errorf("unexpected error registering, got %v", err)
	}

	// updatespec
	// why is this sometimes failing?
	// checkEvalJSON(t, repl, `multiply7.multiply.call(6)`, `42`)
	expNotice := ""
	if repl.lastConfirm != expNotice {
		t.Errorf("incorrect confirmation message: expected %v, got %v", expNotice, repl.lastConfirm)
	}

	// why 0?
	checkEvalJSON(t, repl, `eth.getBlock("pending", true).transactions.length`, `0`)

	txc, repl.xeth = repl.xeth.ApplyTestTxs(repl.stateDb, coinbase, txc)

	checkEvalJSON(t, repl, `admin.contractInfo.start()`, `true`)
	checkEvalJSON(t, repl, `multiply7.multiply.sendTransaction(6, { from: primary, gas: "1000000", gasPrice: "100000" })`, `undefined`)
	expNotice = `About to submit transaction (no NatSpec info found for contract: content hash not found for '0x4a6c99e127191d2ee302e42182c338344b39a37a47cdbb17ab0f26b6802eb4d1'): {"params":[{"to":"0x5dcaace5982778b409c524873b319667eba5d074","data": "0xc6888fa10000000000000000000000000000000000000000000000000000000000000006"}]}`
	if repl.lastConfirm != expNotice {
		t.Errorf("incorrect confirmation message: expected %v, got %v", expNotice, repl.lastConfirm)
	}

	checkEvalJSON(t, repl, `filename = "/tmp/info.json"`, `"/tmp/info.json"`)
	checkEvalJSON(t, repl, `contenthash = admin.contractInfo.register(primary, contractaddress, contract, filename)`, `"0x57e577316ccee6514797d9de9823af2004fdfe22bcfb6e39bbb8f92f57dcc421"`)
	checkEvalJSON(t, repl, `admin.contractInfo.registerUrl(primary, contenthash, "file://"+filename)`, `true`)
	if err != nil {
		t.Errorf("unexpected error registering, got %v", err)
	}

	checkEvalJSON(t, repl, `admin.contractInfo.start()`, `true`)

	// update state
	txc, repl.xeth = repl.xeth.ApplyTestTxs(repl.stateDb, coinbase, txc)

	checkEvalJSON(t, repl, `multiply7.multiply.sendTransaction(6, { from: primary, gas: "1000000", gasPrice: "100000" })`, `undefined`)
	expNotice = "Will multiply 6 by 7."
	if repl.lastConfirm != expNotice {
		t.Errorf("incorrect confirmation message: expected %v, got %v", expNotice, repl.lastConfirm)
	}

}

func checkEvalJSON(t *testing.T, re *testjethre, expr, want string) error {
	val, err := re.re.Run("JSON.stringify(" + expr + ")")
	if err == nil && val.String() != want {
		err = fmt.Errorf("Output mismatch for `%s`:\ngot:  %s\nwant: %s", expr, val.String(), want)
	}
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		file = path.Base(file)
		fmt.Printf("\t%s:%d: %v\n", file, line, err)
		t.Fail()
	}
	return err
}
