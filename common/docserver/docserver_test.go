package docserver

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestGetAuthContent(t *testing.T) {
	text := "test"
	hash := common.Hash{}
	copy(hash[:], crypto.Sha3([]byte(text)))
	ioutil.WriteFile("/tmp/test.content", []byte(text), os.ModePerm)

	ds, err := New("/tmp/")
	content, err := ds.GetAuthContent("file:///test.content", hash)
	if err != nil {
		t.Errorf("no error expected, got %v", err)
	}
	if string(content) != text {
		t.Errorf("incorrect content. expected %v, got %v", text, string(content))
	}

	hash = common.Hash{}
	content, err = ds.GetAuthContent("file:///test.content", hash)
	expected := "content hash mismatch"
	if err == nil {
		t.Errorf("expected error, got nothing")
	} else {
		if err.Error() != expected {
			t.Errorf("expected error '%s' got '%v'", expected, err)
		}
	}

}
