package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer_DeveloperMode(t *testing.T) {

	// get the default config
	config := DefaultConfig()

	// enable developer mode
	config.Developer.Enabled = true
	config.Developer.Period = 2 // block time

	// start the server
	server, err1 := NewServer(config)
	if err1 != nil {
		t.Fatalf("failed to start server: %v", err1)
	}

	// record the initial block number
	blockNumber := server.backend.BlockChain().CurrentBlock().Header().Number.Int64()

	var i int64 = 0
	for i = 0; i < 10; i++ {
		// We expect the node to mine blocks every `config.Developer.Period` time period
		time.Sleep(time.Duration(config.Developer.Period) * time.Second)
		currBlock := server.backend.BlockChain().CurrentBlock().Header().Number.Int64()
		expected := blockNumber + i + 1
		if res := assert.Equal(t, currBlock, expected); res == false {
			break
		}
	}

	// stop the server
	server.Stop()
}
