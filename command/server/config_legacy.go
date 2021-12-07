package server

import (
	"bytes"

	"github.com/naoina/toml"
)

type legacyConfig struct {
	Node struct {
		P2P struct {
			StaticNodes  []string
			TrustedNodes []string
		}
	}
}

func (l *legacyConfig) Config() *Config {
	c := DefaultConfig()
	c.P2P.Discovery.StaticNodes = l.Node.P2P.StaticNodes
	c.P2P.Discovery.TrustedNodes = l.Node.P2P.TrustedNodes
	return c
}

func readLegacyConfig(data []byte) (*Config, error) {
	var legacy legacyConfig

	r := toml.NewDecoder(bytes.NewReader(data))
	if err := r.Decode(&legacy); err != nil {
		return nil, err
	}
	return legacy.Config(), nil
}
