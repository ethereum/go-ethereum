package params

import (
	"bytes"
	"testing"
)

func TestChainConfig_LoadForks(t *testing.T) {
	const config = `
GENESIS_FORK_VERSION: 0x00000000

ALTAIR_FORK_VERSION: 0x00000001
ALTAIR_FORK_EPOCH: 1

EIP7928_FORK_VERSION: 0xb0000038
EIP7928_FORK_EPOCH: 18446744073709551615

BLOB_SCHEDULE: []
`
	c := &ChainConfig{}
	err := c.LoadForks([]byte(config))
	if err != nil {
		t.Fatal(err)
	}

	for _, fork := range c.Forks {
		if fork.Name == "GENESIS" && (fork.Epoch != 0) {
			t.Errorf("unexpected genesis fork epoch %d", fork.Epoch)
		}
		if fork.Name == "ALTAIR" && (fork.Epoch != 1 || !bytes.Equal(fork.Version, []byte{0, 0, 0, 1})) {
			t.Errorf("unexpected altair fork epoch %d version %x", fork.Epoch, fork.Version)
		}
	}
}
