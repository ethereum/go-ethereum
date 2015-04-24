package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"regexp"
	"runtime"
	"strconv"
)

var port = 30300

func testJEthRE(t *testing.T) (*jsre, *eth.Ethereum) {
	tmp, err := ioutil.TempDir("", "geth-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	ks := crypto.NewKeyStorePlain(filepath.Join(tmp, "keys"))
	ethereum, err := eth.New(&eth.Config{
		DataDir:        tmp,
		AccountManager: accounts.NewManager(ks),
		MaxPeers:       0,
		Name:           "test",
	})
	if err != nil {
		t.Fatal("%v", err)
	}
	assetPath := path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist", "assets", "ext")
	repl := newJSRE(ethereum, assetPath, false, "")
	return repl, ethereum
}

func TestNodeInfo(t *testing.T) {
	repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Fatalf("error starting ethereum: %v", err)
	}
	defer ethereum.Stop()

	want := `{"DiscPort":0,"IP":"0.0.0.0","ListenAddr":"","Name":"test","NodeID":"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","NodeUrl":"enode://00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000@0.0.0.0:0","TCPPort":0,"Td":"0"}`
	checkEvalJSON(t, repl, `admin.nodeInfo()`, want)
}

func TestAccounts(t *testing.T) {
	repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Fatalf("error starting ethereum: %v", err)
	}
	defer ethereum.Stop()

	checkEvalJSON(t, repl, `eth.accounts`, `[]`)
	checkEvalJSON(t, repl, `eth.coinbase`, `"0x"`)

	val, err := repl.re.Run(`admin.newAccount("password")`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	addr := val.String()
	if !regexp.MustCompile(`0x[0-9a-f]{40}`).MatchString(addr) {
		t.Errorf("address not hex: %q", addr)
	}

	checkEvalJSON(t, repl, `eth.accounts`, `["`+addr+`"]`)
	checkEvalJSON(t, repl, `eth.coinbase`, `"`+addr+`"`)
}

func TestBlockChain(t *testing.T) {
	repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Fatalf("error starting ethereum: %v", err)
	}
	defer ethereum.Stop()

	// get current block dump before export/import.
	val, err := repl.re.Run("JSON.stringify(admin.debug.dumpBlock())")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	beforeExport := val.String()

	// do the export
	tmp, err := ioutil.TempDir("", "geth-test-export")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	tmpfile := filepath.Join(tmp, "export.chain")
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
	repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Fatalf("error starting ethereum: %v", err)
	}
	defer ethereum.Stop()

	checkEvalJSON(t, repl, `eth.mining`, `false`)
}

func TestRPC(t *testing.T) {
	repl, ethereum := testJEthRE(t)
	if err := ethereum.Start(); err != nil {
		t.Errorf("error starting ethereum: %v", err)
		return
	}
	defer ethereum.Stop()

	checkEvalJSON(t, repl, `admin.startRPC("127.0.0.1", 5004)`, `true`)
}

func checkEvalJSON(t *testing.T, re *jsre, expr, want string) error {
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
