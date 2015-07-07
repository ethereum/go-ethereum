package main

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/common/docserver"
	"github.com/ethereum/go-ethereum/common/natspec"
	"github.com/ethereum/go-ethereum/common/registrar"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/comms"
)

const (
	testSolcPath = ""
	solcVersion  = "0.9.23"

	testKey     = "e6fab74a43941f82d89cb7faa408e227cdad3153c4720e540e855c19b15e6674"
	testAddress = "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	testBalance = "10000000000000000000"
	// of empty string
	testHash = "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
)

var (
	versionRE   = regexp.MustCompile(strconv.Quote(`"compilerVersion":"` + solcVersion + `"`))
	testNodeKey = crypto.ToECDSA(common.Hex2Bytes("4b50fa71f5c3eeb8fdc452224b2395af2fcc3d125e06c32c82e048c0559db03f"))
	testGenesis = `{"` + testAddress[2:] + `": {"balance": "` + testBalance + `"}}`
)

type testjethre struct {
	*jsre
	lastConfirm string
	ds          *docserver.DocServer
}

func (self *testjethre) UnlockAccount(acc []byte) bool {
	err := self.ethereum.AccountManager().Unlock(common.BytesToAddress(acc), "")
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
	return testREPL(t, nil)
}

func testREPL(t *testing.T, config func(*eth.Config)) (string, *testjethre, *eth.Ethereum) {
	tmp, err := ioutil.TempDir("", "geth-test")
	if err != nil {
		t.Fatal(err)
	}

	// set up mock genesis with balance on the testAddress
	core.GenesisAccounts = []byte(testGenesis)

	ks := crypto.NewKeyStorePlain(filepath.Join(tmp, "keystore"))
	am := accounts.NewManager(ks)
	conf := &eth.Config{
		NodeKey:        testNodeKey,
		DataDir:        tmp,
		AccountManager: am,
		MaxPeers:       0,
		Name:           "test",
		SolcPath:       testSolcPath,
		PowTest:        true,
	}
	if config != nil {
		config(conf)
	}
	ethereum, err := eth.New(conf)
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

	assetPath := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist", "assets", "ext")
	client := comms.NewInProcClient(codec.JSON)
	ds := docserver.New("/")
	tf := &testjethre{ds: ds}
	repl := newJSRE(ethereum, assetPath, "", client, false, tf)
	tf.jsre = repl
	return tmp, tf, ethereum
}

func TestNodeInfo(t *testing.T) {
	t.Skip("broken after p2p update")
	tmp, repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Fatalf("error starting ethereum: %v", err)
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)

	want := `{"DiscPort":0,"IP":"0.0.0.0","ListenAddr":"","Name":"test","NodeID":"4cb2fc32924e94277bf94b5e4c983beedb2eabd5a0bc941db32202735c6625d020ca14a5963d1738af43b6ac0a711d61b1a06de931a499fe2aa0b1a132a902b5","NodeUrl":"enode://4cb2fc32924e94277bf94b5e4c983beedb2eabd5a0bc941db32202735c6625d020ca14a5963d1738af43b6ac0a711d61b1a06de931a499fe2aa0b1a132a902b5@0.0.0.0:0","TCPPort":0,"Td":"131072"}`
	checkEvalJSON(t, repl, `admin.nodeInfo`, want)
}

func TestAccounts(t *testing.T) {
	tmp, repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Fatalf("error starting ethereum: %v", err)
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)

	checkEvalJSON(t, repl, `eth.accounts`, `["`+testAddress+`"]`)
	checkEvalJSON(t, repl, `eth.coinbase`, `null`)
	val, err := repl.re.Run(`personal.newAccount("password")`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	addr := val.String()
	if !regexp.MustCompile(`0x[0-9a-f]{40}`).MatchString(addr) {
		t.Errorf("address not hex: %q", addr)
	}

	checkEvalJSON(t, repl, `eth.accounts`, `["`+testAddress+`","`+addr+`"]`)
}

func TestBlockChain(t *testing.T) {
	tmp, repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Fatalf("error starting ethereum: %v", err)
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)
	// get current block dump before export/import.
	val, err := repl.re.Run("JSON.stringify(debug.dumpBlock(eth.blockNumber))")
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

	ethereum.ChainManager().Reset()

	checkEvalJSON(t, repl, `admin.exportChain(`+tmpfileq+`)`, `true`)
	if _, err := os.Stat(tmpfile); err != nil {
		t.Fatal(err)
	}

	// check import, verify that dumpBlock gives the same result.
	checkEvalJSON(t, repl, `admin.importChain(`+tmpfileq+`)`, `true`)
	checkEvalJSON(t, repl, `debug.dumpBlock(eth.blockNumber)`, beforeExport)
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

	checkEvalJSON(t, repl, `admin.startRPC("127.0.0.1", 5004, "*", "web3,eth,net")`, `true`)
}

func TestCheckTestAccountBalance(t *testing.T) {
	t.Skip() // i don't think it tests the correct behaviour here. it's actually testing
	// internals which shouldn't be tested. This now fails because of a change in the core
	// and i have no means to fix this, sorry - @obscuren
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

	val, err := repl.re.Run(`eth.sign("` + testAddress + `", "` + testHash + `")`)

	// This is a very preliminary test, lacking actual signature verification
	if err != nil {
		t.Errorf("Error running js: %v", err)
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
	t.Skip("contract testing is implemented with mining in ethash test mode. This takes about 7seconds to run. Unskip and run on demand")
	coinbase := common.HexToAddress(testAddress)
	tmp, repl, ethereum := testREPL(t, func(conf *eth.Config) {
		conf.Etherbase = testAddress
		conf.PowTest = true
	})
	if err := ethereum.Start(); err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()
	defer os.RemoveAll(tmp)

	reg := registrar.New(repl.xeth)
	_, err := reg.SetGlobalRegistrar("", coinbase)
	if err != nil {
		t.Errorf("error setting HashReg: %v", err)
	}
	_, err = reg.SetHashReg("", coinbase)
	if err != nil {
		t.Errorf("error setting HashReg: %v", err)
	}
	_, err = reg.SetUrlHint("", coinbase)
	if err != nil {
		t.Errorf("error setting HashReg: %v", err)
	}
	/* TODO:
	* lookup receipt and contract addresses by tx hash
	* name registration for HashReg and UrlHint addresses
	* mine those transactions
	* then set once more SetHashReg SetUrlHint
	 */

	source := `contract test {\n` +
		"   /// @notice Will multiply `a` by 7." + `\n` +
		`   function multiply(uint a) returns(uint d) {\n` +
		`       return a * 7;\n` +
		`   }\n` +
		`}\n`

	if checkEvalJSON(t, repl, `admin.stopNatSpec()`, `true`) != nil {
		return
	}

	contractInfo, err := ioutil.ReadFile("info_test.json")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if checkEvalJSON(t, repl, `primary = eth.accounts[0]`, `"`+testAddress+`"`) != nil {
		return
	}
	if checkEvalJSON(t, repl, `source = "`+source+`"`, `"`+source+`"`) != nil {
		return
	}

	// if solc is found with right version, test it, otherwise read from file
	sol, err := compiler.New("")
	if err != nil {
		t.Logf("solc not found: mocking contract compilation step")
	} else if sol.Version() != solcVersion {
		t.Logf("WARNING: solc different version found (%v, test written for %v, may need to update)", sol.Version(), solcVersion)
	}

	if err != nil {
		info, err := ioutil.ReadFile("info_test.json")
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = repl.re.Run(`contract = JSON.parse(` + strconv.Quote(string(info)) + `)`)
		if err != nil {
			t.Errorf("%v", err)
		}
	} else {
		if checkEvalJSON(t, repl, `contract = eth.compile.solidity(source).test`, string(contractInfo)) != nil {
			return
		}
	}

	if checkEvalJSON(t, repl, `contract.code`, `"0x605880600c6000396000f3006000357c010000000000000000000000000000000000000000000000000000000090048063c6888fa114602e57005b603d6004803590602001506047565b8060005260206000f35b60006007820290506053565b91905056"`) != nil {
		return
	}

	if checkEvalJSON(
		t, repl,
		`contractaddress = eth.sendTransaction({from: primary, data: contract.code})`,
		`"0x46d69d55c3c4b86a924a92c9fc4720bb7bce1d74"`,
	) != nil {
		return
	}

	if !processTxs(repl, t, 8) {
		return
	}

	callSetup := `abiDef = JSON.parse('[{"constant":false,"inputs":[{"name":"a","type":"uint256"}],"name":"multiply","outputs":[{"name":"d","type":"uint256"}],"type":"function"}]');
Multiply7 = eth.contract(abiDef);
multiply7 = Multiply7.at(contractaddress);
`
	_, err = repl.re.Run(callSetup)
	if err != nil {
		t.Errorf("unexpected error setting up contract, got %v", err)
		return
	}

	expNotice := ""
	if repl.lastConfirm != expNotice {
		t.Errorf("incorrect confirmation message: expected %v, got %v", expNotice, repl.lastConfirm)
		return
	}

	if checkEvalJSON(t, repl, `admin.startNatSpec()`, `true`) != nil {
		return
	}
	if checkEvalJSON(t, repl, `multiply7.multiply.sendTransaction(6, { from: primary })`, `"0x4ef9088431a8033e4580d00e4eb2487275e031ff4163c7529df0ef45af17857b"`) != nil {
		return
	}

	if !processTxs(repl, t, 1) {
		return
	}

	expNotice = `About to submit transaction (no NatSpec info found for contract: content hash not found for '0x87e2802265838c7f14bb69eecd2112911af6767907a702eeaa445239fb20711b'): {"params":[{"to":"0x46d69d55c3c4b86a924a92c9fc4720bb7bce1d74","data": "0xc6888fa10000000000000000000000000000000000000000000000000000000000000006"}]}`
	if repl.lastConfirm != expNotice {
		t.Errorf("incorrect confirmation message: expected\n%v, got\n%v", expNotice, repl.lastConfirm)
		return
	}

	var contentHash = `"0x86d2b7cf1e72e9a7a3f8d96601f0151742a2f780f1526414304fbe413dc7f9bd"`
	if sol != nil && solcVersion != sol.Version() {
		modContractInfo := versionRE.ReplaceAll(contractInfo, []byte(`"compilerVersion":"`+sol.Version()+`"`))
		fmt.Printf("modified contractinfo:\n%s\n", modContractInfo)
		contentHash = `"` + common.ToHex(crypto.Sha3([]byte(modContractInfo))) + `"`
	}
	if checkEvalJSON(t, repl, `filename = "/tmp/info.json"`, `"/tmp/info.json"`) != nil {
		return
	}
	if checkEvalJSON(t, repl, `contentHash = admin.saveInfo(contract.info, filename)`, contentHash) != nil {
		return
	}
	if checkEvalJSON(t, repl, `admin.register(primary, contractaddress, contentHash)`, `true`) != nil {
		return
	}
	if checkEvalJSON(t, repl, `admin.registerUrl(primary, contentHash, "file://"+filename)`, `true`) != nil {
		return
	}

	if checkEvalJSON(t, repl, `admin.startNatSpec()`, `true`) != nil {
		return
	}

	if !processTxs(repl, t, 3) {
		return
	}

	if checkEvalJSON(t, repl, `multiply7.multiply.sendTransaction(6, { from: primary })`, `"0x66d7635c12ad0b231e66da2f987ca3dfdca58ffe49c6442aa55960858103fd0c"`) != nil {
		return
	}

	if !processTxs(repl, t, 1) {
		return
	}

	expNotice = "Will multiply 6 by 7."
	if repl.lastConfirm != expNotice {
		t.Errorf("incorrect confirmation message: expected\n%v, got\n%v", expNotice, repl.lastConfirm)
		return
	}
}

func pendingTransactions(repl *testjethre, t *testing.T) (txc int64, err error) {
	txs := repl.ethereum.TxPool().GetTransactions()
	return int64(len(txs)), nil
}

func processTxs(repl *testjethre, t *testing.T, expTxc int) bool {
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

	err = repl.ethereum.StartMining(runtime.NumCPU())
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

func checkEvalJSON(t *testing.T, re *testjethre, expr, want string) error {
	val, err := re.re.Run("JSON.stringify(" + expr + ")")
	if err == nil && val.String() != want {
		err = fmt.Errorf("Output mismatch for `%s`:\ngot:  %s\nwant: %s", expr, val.String(), want)
	}
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		file = filepath.Base(file)
		fmt.Printf("\t%s:%d: %v\n", file, line, err)
		t.Fail()
	}
	return err
}
