package javascript

import (
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
)

func TestJEthRE(t *testing.T) {
	ethereum, err := eth.New(&eth.Config{
		DataDir:  "/tmp/eth",
		KeyStore: "file",
	})
	// os.WriteFile("/tmp/eth/default.addr", []byte("25ec29286951d5acc52a4f4d631f479c1002f97"))
	ethutil.WriteFile("/tmp/eth/default.prv", []byte("946d0fe6dd95ef5476dd1fd204626b59c973bd72ffac4a108827ef488465cc68"))

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
}
