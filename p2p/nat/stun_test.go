package nat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNatStun(t *testing.T) {
	nat, err := newSTUN("default")
	assert.NoError(t, err)
	_, err = nat.ExternalIP()
	assert.NoError(t, err)
}

func TestUnreachedNatServer(t *testing.T) {
	stun := &stun{
		serverList: []string{"198.51.100.2:1234", "198.51.100.5"},
	}
	_, err := stun.ExternalIP()
	if err != errSTUNFailed {
		t.Fatal("wrong error:", err)
	}
}
