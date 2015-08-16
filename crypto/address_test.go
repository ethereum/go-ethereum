package crypto

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestChecksum(t *testing.T) {
	var addr common.Address
	addr[0] = 1

	checksumed := ChecksumAddress(addr)
	if !bytes.Equal(checksumed.Checksum[:], Sha3(checksumed.Address[:])[:4]) {
		t.Error("checksum failed")
	}

	addr, err := ValidateAddress(checksumed)
	if err != nil {
		t.Error(err)
	}

	if addr != addr {
		t.Error("address failed")
	}

	checksumed.Checksum[3] |= 1
	_, err = ValidateAddress(checksumed)
	if err == nil {
		t.Error("checksum success, should have failed")
	}
}
