package types

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"io/ioutil"
	"log"
	"testing"
)

func TestBALEncoding(t *testing.T) {
	b := NewBlockAccessList()
	// TODO: populate the BAL with something
	var buf bytes.Buffer
	err := b.EncodeRLP(&buf)
	if err != nil {
		t.Fatalf("encoding failed: %v\n", err)
	}
	if err := b.DecodeRLP(rlp.NewStream(bytes.NewReader(buf.Bytes()), 10000000)); err != nil {
		t.Fatalf("decoding failed: %v\n", err)
	}
}

func TestBALEncodingPython(t *testing.T) {
	var encObj encodingBlockAccessList
	filename := "testdata/22400032_block_access_list.txt"

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	bytes, err := hex.DecodeString(string(data))
	if err != nil {
		log.Fatalf("Failed to decode hex string: %v", err)
	}

	fmt.Printf("Decoded bytes: %x\n", bytes)
	if err := encObj.UnmarshalSSZ(bytes); err != nil {
		t.Fatalf("error unmarshalling ssz: %v", err)
	}
}

func TestBALBlockEncoding(t *testing.T) {
	header := Header{}
	bal := NewBlockAccessList()
	block := NewBlock(&header, &Body{nil, nil, nil, bal}, nil, nil)
	buf, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatalf("encoding failed: %v\n", err)
	}
	fmt.Printf("buf is %x\n", buf)
	if err := block.DecodeRLP(rlp.NewStream(bytes.NewReader(buf), 10000000)); err != nil {
		t.Fatalf("decoding failed: %v\n", err)
	}
}

func TestBALCreation(t *testing.T) {
	// test ideas (we will want these as unit tests first, and then state tests later on):
	// * contract creates another contract.  the creator was itself created in the same transaction
	// * contract is created in the current transaction, creates another contract, and then self-destructs
	// * create/create2 deployment reverts.  Ensure the creator and the target nonces are included
}
