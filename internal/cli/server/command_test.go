package server

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFlags(t *testing.T) {
	t.Parallel()

	var c Command

	args := []string{
		"--txpool.rejournal", "30m0s",
		"--txpool.lifetime", "30m0s",
		"--miner.gasprice", "20000000000",
		"--gpo.maxprice", "70000000000",
		"--gpo.ignoreprice", "1",
		"--cache.trie.rejournal", "40m0s",
		"--dev",
		"--dev.period", "2",
		"--datadir", "./data",
		"--maxpeers", "30",
		"--requiredblocks", "a=b",
		"--http.api", "eth,web3,bor",
	}
	err := c.extractFlags(args)

	require.NoError(t, err)

	txRe, _ := time.ParseDuration("30m0s")
	txLt, _ := time.ParseDuration("30m0s")
	caRe, _ := time.ParseDuration("40m0s")

	require.Equal(t, c.config.DataDir, "./data")
	require.Equal(t, c.config.Developer.Enabled, true)
	require.Equal(t, c.config.Developer.Period, uint64(2))
	require.Equal(t, c.config.TxPool.Rejournal, txRe)
	require.Equal(t, c.config.TxPool.LifeTime, txLt)
	require.Equal(t, c.config.Sealer.GasPrice, big.NewInt(20000000000))
	require.Equal(t, c.config.Gpo.MaxPrice, big.NewInt(70000000000))
	require.Equal(t, c.config.Gpo.IgnorePrice, big.NewInt(1))
	require.Equal(t, c.config.Cache.Rejournal, caRe)
	require.Equal(t, c.config.P2P.MaxPeers, uint64(30))
	require.Equal(t, c.config.RequiredBlocks, map[string]string{"a": "b"})
	require.Equal(t, c.config.JsonRPC.Http.API, []string{"eth", "web3", "bor"})
}
