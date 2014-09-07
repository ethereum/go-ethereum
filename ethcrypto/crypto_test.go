package ethcrypto

import (
	"bytes"
	"testing"

	"github.com/ethereum/eth-go/ethutil"
)

func TestSha3(t *testing.T) {
	const exp = "3a985da74fe225b2045c172d6bd390bd855f086e3e9d525b46bfe24511431532"
	sha3_256 := Sha3Bin([]byte("abc"))
	if bytes.Compare(sha3_256, ethutil.Hex2Bytes(exp)) != 0 {
		t.Errorf("Sha3_256 failed. Incorrect result %x", sha3_256)
	}
}
