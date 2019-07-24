package simulation

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestExecAdapter(t *testing.T) {

	execPath := "../build/bin/swarm"

	// Skip test if binary doesn't exist
	if _, err := os.Stat(execPath); err != nil {
		if os.IsNotExist(err) {
			t.Skip("swarm binary not found. build it before running the test")
		}
	}

	tmpdir, err := ioutil.TempDir("", "test-adapter-exec")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	adapter, err := NewExecAdapter(ExecAdapterConfig{
		ExecutablePath:    execPath,
		BaseDataDirectory: tmpdir,
	})
	if err != nil {
		t.Fatalf("could not create exec adapter: %v", err)
	}

	bzzkey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("could not generate key: %v", err)
	}
	bzzkeyhex := hex.EncodeToString(crypto.FromECDSA(bzzkey))

	nodekey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("could not generate key: %v", err)
	}
	nodekeyhex := hex.EncodeToString(crypto.FromECDSA(nodekey))

	args := []string{
		"--bootnodes", "",
		"--bzzkeyhex", bzzkeyhex,
		"--nodekeyhex", nodekeyhex,
		"--bzznetworkid", "499",
	}
	nodeconfig := NodeConfig{
		ID:     "node1",
		Args:   args,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	node := adapter.NewNode(nodeconfig)
	info := node.Info()
	if info.ID != "node1" {
		t.Fatal("node id is different")
	}

	err = node.Start()
	if err != nil {
		t.Fatalf("node did not start: %v", err)
	}

	infoA := node.Info()

	err = node.Stop()
	if err != nil {
		t.Fatalf("node didn't stop: %v", err)
	}

	err = node.Start()
	if err != nil {
		t.Fatalf("node didn't start again: %v", err)
	}

	infoB := node.Info()

	if infoA.BzzAddr != infoB.BzzAddr {
		t.Errorf("bzzaddr should be the same: %s - %s", infoA.Enode, infoB.Enode)
	}

	err = node.Stop()
	if err != nil {
		t.Fatalf("node didn't stop: %v", err)
	}
}
