package cli

import (
	"testing"

	"github.com/mitchellh/cli"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/internal/cli/server"
)

func TestStatusCommand(t *testing.T) {
	t.Parallel()

	// Start a blockchain in developer
	config := server.DefaultConfig()

	// enable developer mode
	config.Developer.Enabled = true
	config.Developer.Period = 2

	// start the mock server
	srv, err := server.CreateMockServer(config)
	require.NoError(t, err)

	defer server.CloseMockServer(srv)

	// get the grpc port
	port := srv.GetGrpcAddr()

	command1 := &StatusCommand{
		Meta2: &Meta2{
			UI:   cli.NewMockUi(),
			addr: "127.0.0.1:" + port,
		},
		wait: true,
	}

	status := command1.Run([]string{"-w", "--address", command1.Meta2.addr})

	require.Equal(t, 0, status)
}
