package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer_DeveloperMode(t *testing.T) {
	t.Parallel()

	// get the default config
	config := DefaultConfig()

	// enable developer mode
	config.Developer.Enabled = true
	config.Developer.Period = 2 // block time

	// start the mock server
	server, err := CreateMockServer(config)
	assert.NoError(t, err)

	defer CloseMockServer(server)

	// record the initial block number
	blockNumber := server.backend.BlockChain().CurrentBlock().Number.Int64()

	var i int64
	for i = 0; i < 3; i++ {
		// We expect the node to mine blocks every `config.Developer.Period` time period
		time.Sleep(time.Duration(config.Developer.Period) * time.Second)

		currBlock := server.backend.BlockChain().CurrentBlock().Number.Int64()
		expected := blockNumber + i + 1

		if res := assert.Equal(t, currBlock, expected); res == false {
			break
		}
	}
}
