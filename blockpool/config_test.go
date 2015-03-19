package blockpool

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/blockpool/test"
)

func TestBlockPoolConfig(t *testing.T) {
	test.LogInit()
	blockPool := &BlockPool{Config: &Config{}}
	blockPool.Start()
	c := blockPool.Config
	test.CheckInt("BlockHashesBatchSize", c.BlockHashesBatchSize, blockHashesBatchSize, t)
	test.CheckInt("BlockBatchSize", c.BlockBatchSize, blockBatchSize, t)
	test.CheckInt("BlocksRequestRepetition", c.BlocksRequestRepetition, blocksRequestRepetition, t)
	test.CheckInt("BlocksRequestMaxIdleRounds", c.BlocksRequestMaxIdleRounds, blocksRequestMaxIdleRounds, t)
	test.CheckDuration("BlockHashesRequestInterval", c.BlockHashesRequestInterval, blockHashesRequestInterval, t)
	test.CheckDuration("BlocksRequestInterval", c.BlocksRequestInterval, blocksRequestInterval, t)
	test.CheckDuration("BlockHashesTimeout", c.BlockHashesTimeout, blockHashesTimeout, t)
	test.CheckDuration("BlocksTimeout", c.BlocksTimeout, blocksTimeout, t)
	test.CheckDuration("IdleBestPeerTimeout", c.IdleBestPeerTimeout, idleBestPeerTimeout, t)
	test.CheckDuration("PeerSuspensionInterval", c.PeerSuspensionInterval, peerSuspensionInterval, t)
}

func TestBlockPoolOverrideConfig(t *testing.T) {
	test.LogInit()
	blockPool := &BlockPool{Config: &Config{}}
	c := &Config{128, 32, 1, 0, 300 * time.Millisecond, 100 * time.Millisecond, 90 * time.Second, 0, 30 * time.Second, 30 * time.Second}

	blockPool.Config = c
	blockPool.Start()
	test.CheckInt("BlockHashesBatchSize", c.BlockHashesBatchSize, 128, t)
	test.CheckInt("BlockBatchSize", c.BlockBatchSize, 32, t)
	test.CheckInt("BlocksRequestRepetition", c.BlocksRequestRepetition, blocksRequestRepetition, t)
	test.CheckInt("BlocksRequestMaxIdleRounds", c.BlocksRequestMaxIdleRounds, blocksRequestMaxIdleRounds, t)
	test.CheckDuration("BlockHashesRequestInterval", c.BlockHashesRequestInterval, 300*time.Millisecond, t)
	test.CheckDuration("BlocksRequestInterval", c.BlocksRequestInterval, 100*time.Millisecond, t)
	test.CheckDuration("BlockHashesTimeout", c.BlockHashesTimeout, 90*time.Second, t)
	test.CheckDuration("BlocksTimeout", c.BlocksTimeout, blocksTimeout, t)
	test.CheckDuration("IdleBestPeerTimeout", c.IdleBestPeerTimeout, 30*time.Second, t)
	test.CheckDuration("PeerSuspensionInterval", c.PeerSuspensionInterval, 30*time.Second, t)
}
