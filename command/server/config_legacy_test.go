package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigLegacy(t *testing.T) {
	toml := `[Node.P2P]
StaticNodes = ["node1"]
TrustedNodes = ["node2"]`

	config, err := readLegacyConfig([]byte(toml))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, config.P2P.Discovery.StaticNodes, []string{"node1"})
	assert.Equal(t, config.P2P.Discovery.TrustedNodes, []string{"node2"})
}
