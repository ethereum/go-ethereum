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
		serverList: []string{"1.2.3.4:1234", "1.2.3.4:1234", "1.2.3.4:1234"},
	}
	stun.serverList = append(stun.serverList, stunDefaultServerList...)
	_, err := stun.ExternalIP()
	assert.NoError(t, err)
}
