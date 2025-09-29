package types

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
	"os"
	"testing"
)

func TestBALDecoding(t *testing.T) {
	var (
		err  error
		data []byte
	)
	data, err = os.ReadFile("blocks_bal_one.rlp")
	if err != nil {
		t.Fatalf("error opening file: %v", err)
	}
	reader := bytes.NewReader(data)
	stream := rlp.NewStream(reader, 0)
	var blocks Block
	for i := 0; err == nil; i++ {
		fmt.Printf("decode %d\n", i)
		err = stream.Decode(&blocks)
		if err != nil && err != io.EOF {
			t.Fatalf("error decoding blocks: %v", err)
		}
		fmt.Printf("block number is %d\n", blocks.NumberU64())
	}
}
