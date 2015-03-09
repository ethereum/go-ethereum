package javascript

import (
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
)

func TestJEthRE(t *testing.T) {

	err := os.MkdirAll("/tmp/eth/data/", os.ModePerm)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	ethereum, err := eth.New(&eth.Config{
		DataDir:  "/tmp/eth",
		KeyStore: "file",
	})
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	err = ethutil.WriteFile("/tmp/eth/default.prv", []byte("946d0fe6dd95ef5476dd1fd204626b59c973bd72ffac4a108827ef488465cc68"))
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	t.Logf("ethereum %#v", ethereum)
	assetPath := path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist", "assets", "ext")
	jethre := NewJEthRE(ethereum, assetPath)

	val, err := jethre.Run("eth.getCoinbase()")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	pp, err := jethre.PrettyPrint(val)
	if err != nil {
		t.Errorf("%v", err)
	}

	if !val.IsString() {
		t.Errorf("incorrect type, expected string, got %v: %v", val, pp)
	}
	strVal, _ := val.ToString()
	expected := "0x25ec29286951d5acc52a4f4d631f479c1002f97b"
	if strVal != expected {
		t.Errorf("incorrect result, expected %s, got %v", expected, strVal)
	}

	// should get current block
	// val, err = jethre.Run("eth.getBlock()")
	// if err != nil {
	// 	t.Errorf("expected no error, got %v", err)
	// }

	fn := "/tmp/eth/data/blockchain.0"
	val, err = jethre.Run("eth.export(\"" + fn + "\")")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if _, err = os.Stat(fn); err != nil {
		t.Errorf("expected no error on file, got %v", err)
	}

	ethereum, err = eth.New(&eth.Config{
		DataDir:  "/tmp/eth1",
		KeyStore: "file",
	})
	val, err = jethre.Run("eth.import(\"" + fn + "\")")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// var val0 otto.Value

	// should get current block
	// val0, err = jethre.Run("eth.getBlock()")
	// if err != nil {
	// 	t.Errorf("expected no error, got %v", err)
	// }

	// if v0 != v1 {
	// 	t.Errorf("expected same head after export-import, got %v (!=%v)", v1, v0)
	// }

	ethereum.Start()
	// FIXME:
	// ethereum.Stop causes panic: runtime error: invalid memory address or nil pointer
	// github.com/ethereum/go-ethereum/eth.(*Ethereum).Stop(0xc208f46270)
	//         /Users/tron/Work/ethereum/go/src/github.com/ethereum/go-ethereum/eth/backend.go:292 +0xdc
	// defer ethereum.Stop()

	val, err = jethre.Run("eth.isMining()")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	var mining bool
	mining, err = val.ToBoolean()
	if err != nil {
		t.Errorf("expected boolean, got %v", err)
	}
	if mining {
		t.Errorf("expected false (not mining), got true")
	}

	val, err = jethre.Run("eth.setMining(true)")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	mining, _ = val.ToBoolean()
	if !mining {
		t.Errorf("expected true (mining), got false")
	}
	val, err = jethre.Run("eth.isMining()")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	mining, err = val.ToBoolean()
	if err != nil {
		t.Errorf("expected boolean, got %v", err)
	}
	if !mining {
		t.Errorf("expected true (mining), got false")
	}

	val, err = jethre.Run("eth.setMining(true)")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	mining, _ = val.ToBoolean()
	if !mining {
		t.Errorf("expected true (mining), got false")
	}

	val, err = jethre.Run("eth.setMining(false)")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	mining, _ = val.ToBoolean()
	if mining {
		t.Errorf("expected false (not mining), got true")
	}
}
