package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer_DeveloperMode(t *testing.T) {
	// TODO: As developer mode uses clique consensus, block production might not work
	// directly. We need some workaround to get the dev mode work.
	t.Skip("TODO: Skipping tests as dev mode is not working as expected")
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

		if res := assert.Equal(t, expected, currBlock); res == false {
			break
		}
	}
}
