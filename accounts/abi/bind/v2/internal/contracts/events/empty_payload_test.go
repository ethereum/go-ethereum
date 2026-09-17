package events

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestUnpackBasic1EmptyPayload(t *testing.T) {
	c := NewC()
	log := &types.Log{
		Topics: []common.Hash{
			c.abi.Events["basic1"].ID,
			common.BigToHash(big.NewInt(1)),
		},
		Data: nil,
	}
	_, err := c.UnpackBasic1Event(log)
	if err == nil {
		t.Fatal("expected error unpacking empty payload for non-indexed fields")
	}
	if !strings.Contains(err.Error(), "empty string") && !strings.Contains(err.Error(), "abi:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnpackBasic1ValidPayload(t *testing.T) {
	c := NewC()
	// ABI-encode uint256(2) as non-indexed data
	data := common.LeftPadBytes(big.NewInt(2).Bytes(), 32)
	log := &types.Log{
		Topics: []common.Hash{
			c.abi.Events["basic1"].ID,
			common.BigToHash(big.NewInt(1)),
		},
		Data: data,
	}
	out, err := c.UnpackBasic1Event(log)
	if err != nil {
		t.Fatal(err)
	}
	if out.Id.Cmp(big.NewInt(1)) != 0 || out.Data.Cmp(big.NewInt(2)) != 0 {
		t.Fatalf("got id=%v data=%v", out.Id, out.Data)
	}
}
