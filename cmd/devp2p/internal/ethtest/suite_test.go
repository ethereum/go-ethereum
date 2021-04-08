package ethtest

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

var node = "enode://04eab89943b5b2aa38745051471b31dc09d187c99b8fc6b740a8587ac43d1d65d644086a862189bc6298198453ba68076ebdb9cd7013d8893878f628a664376e@127.0.0.1:30303"

func TestStatusGeth(t *testing.T) {
	geth, datadir := initGeth(t)
	go geth.Run()
	defer func() {
		if geth.Process != nil {
			kill(geth)
		}
		os.RemoveAll(datadir)
	}()

	// wait for geth to start up
	time.Sleep(time.Second)

	suite := newTestSuite(t)
	failed, output := utesting.Run(utesting.Test{"TestStatus", suite.TestStatus})
	if failed {
		t.Fatalf("test failed: \n%s", output)
	}
}

func TestHandshakeGeth(t *testing.T) {
	datadir := tmpdir(t)
	filepaths, err := filepaths()
	if err != nil {
		t.Fatalf("could not get paths for init files: %v", err)
	}

	args := []string{"--datadir", datadir, "--nodiscover", "--nat=none", "--networkid=19763", "--nodekey", filepaths["nodekey"], "--verbosity=5"}
	geth := runGeth(args)
	go func() {
		if err := geth.Run(); err != nil {
			os.RemoveAll(datadir)
			t.Fatalf("could not run geth: %v", err)
		}
	}()
	defer func() {
		kill(geth)
		os.RemoveAll(datadir)
	}()

	// wait for geth to start up
	time.Sleep(time.Second)

	suite := newTestSuite(t)
	_, err = suite.dial()
	if err != nil {
		t.Fatalf("could not handshake with node: %v", err)
	}
}

func newTestSuite(t *testing.T) *Suite {
	filepaths, err := filepaths()
	if err != nil {
		t.Fatalf("could not get paths for files: %v", err)
	}

	var (
		id    *enode.Node
		suite *Suite
	)

	id, err = enode.ParseV4(node)
	if err != nil {
		t.Fatalf("could not get enode: %v", err)
	}
	suite, err = NewSuite(id, filepaths["fullchain"], filepaths["genesis"])
	if err != nil {
		t.Fatalf("could not create test suite: %v", err)
	}
	return suite
}

func filepaths() (map[string]string, error)  {
	files := make(map[string]string)
	var (
		genesis, halfchain, fullchain, nodekey string
		err                                    error
	)

	genesis, err = filepath.Abs("./testdata/genesis.json")
	if err != nil {
		return files, err
	}
	files["genesis"] = genesis

	halfchain, err = filepath.Abs("./testdata/halfchain.rlp")
	if err != nil {
		return files, err
	}
	files["halfchain"] = halfchain

	fullchain, err = filepath.Abs("./testdata/chain.rlp")
	if err != nil {
		return files, err
	}
	files["fullchain"] = fullchain

	nodekey, err = filepath.Abs("./testdata/nodekey")
	if err != nil {
		return files, err
	}
	files["nodekey"] = nodekey

	return files, nil
}

func initGeth(t *testing.T) (*exec.Cmd, string) {
	datadir := tmpdir(t)
	filepaths, err := filepaths()
	if err != nil {
		t.Fatalf("could not get paths for init files: %v", err)
	}

	initArgs := []string{"init", filepaths["genesis"], "--datadir", datadir}
	initCmd := exec.Command("./geth", initArgs...)
	if err := initCmd.Run(); err != nil {
		t.Fatalf("could not run command: %v", err)
	}

	importArgs := []string{"import", filepaths["halfchain"], "--datadir", datadir}
	importCmd := exec.Command("./geth", importArgs...)
	if err := importCmd.Run(); err != nil {
		t.Fatalf("could not run command: %v", err)
	}

	runArgs := []string{"--datadir", datadir, "--nodiscover", "--nat=none", "--networkid=19763", "--nodekey", filepaths["nodekey"], "--verbosity=5"} // TODO
	return runGeth(runArgs), datadir
}

func runGeth(args []string) *exec.Cmd {
	runCmd := exec.Command("./geth", args...)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	return runCmd
}

func kill(cmd *exec.Cmd) {
	cmd.Process.Kill()
}

func tmpdir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "geth-test")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
