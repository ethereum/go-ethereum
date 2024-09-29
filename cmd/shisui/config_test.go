package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestGenConfig(t *testing.T) {
	size := uint64(5 * 1000 * 1000 * 1000)
	flagSet := flag.NewFlagSet("test", 0)
	flagSet.String("rpc.addr", "127.0.0.11", "test")
	flagSet.String("rpc.port", "8888", "test")
	tmpDir := t.TempDir()
	flagSet.String("data.dir", tmpDir, "test")
	flagSet.Uint64("data.capacity", size, "test")
	// flagSet.String("udp.addr", "172.23.50.11", "test")
	flagSet.Int("udp.port", 9999, "test")
	flagSet.Int("loglevel", 3, "test")
	val := cli.NewStringSlice("history")
	flagSet.Var(val, "networks", "test")

	command := &cli.Command{Name: "mycommand"}

	ctx := cli.NewContext(nil, flagSet, nil)
	ctx.Command = command

	config, err := getPortalConfig(ctx)
	require.NoError(t, err)

	require.Equal(t, config.DataCapacity, size)
	require.Equal(t, config.DataDir, tmpDir)
	require.Equal(t, config.LogLevel, 3)
	// require.Equal(t, config.RpcAddr, "127.0.0.11:8888")
	require.Equal(t, config.Protocol.ListenAddr, ":9999")
	require.Equal(t, config.Networks, []string{"history"})
}

func TestKeyConfig(t *testing.T) {
	flagSet := flag.NewFlagSet("test", 0)
	tmpDir := t.TempDir()
	flagSet.String("data.dir", tmpDir, "test")
	pk := "a19d7a264e68004832327fca0ac46636332e0ec4b2a20a7ac942020754fcb666"
	flagSet.String("private.key", "0x"+pk, "test")

	command := &cli.Command{Name: "mycommand"}

	ctx := cli.NewContext(nil, flagSet, nil)
	ctx.Command = command

	config, err := getPortalConfig(ctx)
	require.NoError(t, err)

	require.Equal(t, config.DataDir, tmpDir)

	keyPk, err := crypto.HexToECDSA(pk)
	require.Nil(t, err)
	require.Equal(t, config.PrivateKey, keyPk)

	fullPath := filepath.Join(config.DataDir, privateKeyFileName)
	keyStored, err := os.ReadFile(fullPath)
	require.Nil(t, err)
	keyEnc := string(keyStored)
	require.Equal(t, keyEnc, pk)
}
