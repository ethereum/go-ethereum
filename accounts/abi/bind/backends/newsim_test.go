package backends

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/core"
)

func TestNewSim(t *testing.T) {
	genAlloc := make(core.GenesisAlloc)
	newSim, err := NewNewSim(genAlloc)
	if err != nil {
		t.Fatal(err)
	}
	num, err := newSim.Client.BlockNumber(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if num != 0 {
		t.Fatalf("expected 0 got %v", num)
	}
	// Create a block
	newSim.Commit()
	num, err = newSim.Client.BlockNumber(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if num != 1 {
		t.Fatalf("expected 1 got %v", num)
	}
}
