package server

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestFlagsWithoutConfig tests all types of flags passed only
// via cli args.
func TestFlagsWithoutConfig(t *testing.T) {
	t.Parallel()

	var c Command

	args := []string{
		"--identity", "",
		"--datadir", "./data",
		"--verbosity", "3",
		"--rpc.batchlimit", "0",
		"--snapshot",
		"--bor.logs=false",
		"--eth.requiredblocks", "a=b",
		"--miner.gasprice", "30000000000",
		"--miner.recommit", "20s",
		"--rpc.evmtimeout", "5s",
		"--rpc.txfeecap", "6.0",
		"--http.api", "eth,bor",
		"--ws.api", "",
		"--gpo.maxprice", "5000000000000",
	}

	err := c.extractFlags(args)

	require.NoError(t, err)

	recommit, _ := time.ParseDuration("20s")
	evmTimeout, _ := time.ParseDuration("5s")

	require.Equal(t, c.config.Identity, "")
	require.Equal(t, c.config.DataDir, "./data")
	require.Equal(t, c.config.KeyStoreDir, "")
	require.Equal(t, c.config.Verbosity, 3)
	require.Equal(t, c.config.RPCBatchLimit, uint64(0))
	require.Equal(t, c.config.Snapshot, true)
	require.Equal(t, c.config.BorLogs, false)
	require.Equal(t, c.config.RequiredBlocks, map[string]string{"a": "b"})
	require.Equal(t, c.config.Sealer.GasPrice, big.NewInt(30000000000))
	require.Equal(t, c.config.Sealer.Recommit, recommit)
	require.Equal(t, c.config.JsonRPC.RPCEVMTimeout, evmTimeout)
	require.Equal(t, c.config.JsonRPC.Http.API, []string{"eth", "bor"})
	require.Equal(t, c.config.JsonRPC.Ws.API, []string(nil))
	require.Equal(t, c.config.Gpo.MaxPrice, big.NewInt(5000000000000))
}

// TestFlagsWithoutConfig tests all types of flags passed only
// via config file.
func TestFlagsWithConfig(t *testing.T) {
	t.Parallel()

	var c Command

	args := []string{
		"--config", "./testdata/test.toml",
	}

	err := c.extractFlags(args)

	require.NoError(t, err)

	recommit, _ := time.ParseDuration("20s")
	evmTimeout, _ := time.ParseDuration("5s")

	require.Equal(t, c.config.Identity, "")
	require.Equal(t, c.config.DataDir, "./data")
	require.Equal(t, c.config.KeyStoreDir, "./keystore")
	require.Equal(t, c.config.Verbosity, 3)
	require.Equal(t, c.config.RPCBatchLimit, uint64(0))
	require.Equal(t, c.config.Snapshot, true)
	require.Equal(t, c.config.BorLogs, false)
	require.Equal(t, c.config.RequiredBlocks,
		map[string]string{
			"31000000": "0x2087b9e2b353209c2c21e370c82daa12278efd0fe5f0febe6c29035352cf050e",
			"32000000": "0x875500011e5eecc0c554f95d07b31cf59df4ca2505f4dbbfffa7d4e4da917c68",
		},
	)
	require.Equal(t, c.config.Sealer.GasPrice, big.NewInt(25000000000))
	require.Equal(t, c.config.Sealer.Recommit, recommit)
	require.Equal(t, c.config.JsonRPC.RPCEVMTimeout, evmTimeout)
	require.Equal(t, c.config.JsonRPC.Http.API, []string{"eth", "bor"})
	require.Equal(t, c.config.JsonRPC.Ws.API, []string{""})
	require.Equal(t, c.config.Gpo.MaxPrice, big.NewInt(5000000000000))
}

// TestFlagsWithConfig tests all types of flags passed via both
// config file and cli args. The cli args should overwrite the
// value of flag.
func TestFlagsWithConfigAndFlags(t *testing.T) {
	t.Parallel()

	var c Command

	// Set the config and also override
	args := []string{
		"--config", "./testdata/test.toml",
		"--identity", "Anon",
		"--datadir", "",
		"--keystore", "",
		"--verbosity", "0",
		"--rpc.batchlimit", "5",
		"--snapshot=false",
		"--bor.logs=true",
		"--eth.requiredblocks", "x=y",
		"--miner.gasprice", "60000000000",
		"--miner.recommit", "30s",
		"--rpc.evmtimeout", "0s",
		"--rpc.txfeecap", "0",
		"--http.api", "",
		"--ws.api", "eth,bor,web3",
		"--gpo.maxprice", "0",
	}

	err := c.extractFlags(args)

	require.NoError(t, err)

	recommit, _ := time.ParseDuration("30s")
	evmTimeout, _ := time.ParseDuration("0s")

	require.Equal(t, c.config.Identity, "Anon")
	require.Equal(t, c.config.DataDir, "")
	require.Equal(t, c.config.KeyStoreDir, "")
	require.Equal(t, c.config.Verbosity, 0)
	require.Equal(t, c.config.RPCBatchLimit, uint64(5))
	require.Equal(t, c.config.Snapshot, false)
	require.Equal(t, c.config.BorLogs, true)
	require.Equal(t, c.config.RequiredBlocks, map[string]string{"x": "y"})
	require.Equal(t, c.config.Sealer.GasPrice, big.NewInt(60000000000))
	require.Equal(t, c.config.Sealer.Recommit, recommit)
	require.Equal(t, c.config.JsonRPC.RPCEVMTimeout, evmTimeout)
	require.Equal(t, c.config.JsonRPC.Http.API, []string(nil))
	require.Equal(t, c.config.JsonRPC.Ws.API, []string{"eth", "bor", "web3"})
	require.Equal(t, c.config.Gpo.MaxPrice, big.NewInt(0))
}
