package main

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestGenConfig(t *testing.T) {
	size := uint64(5 * 1000 * 1000 * 1000)
	flagSet := flag.NewFlagSet("test", 0)
	flagSet.String("rpc.addr", "127.0.0.11", "test")
	flagSet.String("rpc.port", "8888", "test")
	flagSet.String("data.dir", "./test", "test")
	flagSet.Uint64("data.capacity", size, "test")
	flagSet.String("udp.addr", "172.23.50.11", "test")
	flagSet.Int("udp.port", 9999, "test")
	flagSet.Int("loglevel", 3, "test")

	command := &cli.Command{Name: "mycommand"}

	ctx := cli.NewContext(nil, flagSet, nil)
	ctx.Command = command

	config, err := getPortalHistoryConfig(ctx)
	require.NoError(t, err)

	require.Equal(t, config.DataCapacity, size)
	require.Equal(t, config.DataDir, "./test")
	require.Equal(t, config.LogLevel, 3)
	require.Equal(t, config.RpcAddr, "127.0.0.11:8888")
	require.Equal(t, config.Protocol.ListenAddr, ":9999")
}
