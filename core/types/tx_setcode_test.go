package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestParseDelegation(t *testing.T) {
	addr := common.Address{0x42}
	d := append(DelegationPrefix, addr.Bytes()...)
	if got, ok := ParseDelegation(d); !ok || addr != got {
		t.Fatalf("failed to parse, got %s %v", got.Hex(), ok)
	}
}
