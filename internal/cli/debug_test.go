package cli

import (
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/mitchellh/cli"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/internal/cli/server"
)

var currentDir string

func TestCommand_DebugBlock(t *testing.T) {
	t.Parallel()

	// Start a blockchain in developer mode and get trace of block
	config := server.DefaultConfig()

	// enable developer mode
	config.Developer.Enabled = true
	config.Developer.Period = 2          // block time
	config.Developer.GasLimit = 11500000 // initial block gaslimit

	// enable archive mode for getting traces of ancient blocks
	config.GcMode = "archive"

	// start the mock server
	srv, err := server.CreateMockServer(config)
	require.NoError(t, err)

	defer server.CloseMockServer(srv)

	// get the grpc port
	port := srv.GetGrpcAddr()

	// wait for 4 seconds to mine a 2 blocks
	time.Sleep(2 * time.Duration(config.Developer.Period) * time.Second)

	// add prefix for debug trace
	prefix := "bor-block-trace-"

	// output dir
	output := "debug_block_test"

	// set current directory
	currentDir, _ = os.Getwd()

	// trace 1st block
	start := time.Now()
	dst1 := path.Join(output, prefix+time.Now().UTC().Format("2006-01-02-150405Z"), "block.json")
	res := traceBlock(port, 1, output)
	require.Equal(t, 0, res)
	t.Logf("Completed trace of block %d in %d ms at %s", 1, time.Since(start).Milliseconds(), dst1)

	// adding this to avoid debug directory name conflicts
	time.Sleep(time.Second)

	// trace last/recent block
	start = time.Now()
	latestBlock := srv.GetLatestBlockNumber().Int64()
	dst2 := path.Join(output, prefix+time.Now().UTC().Format("2006-01-02-150405Z"), "block.json")
	res = traceBlock(port, latestBlock, output)
	require.Equal(t, 0, res)
	t.Logf("Completed trace of block %d in %d ms at %s", latestBlock, time.Since(start).Milliseconds(), dst2)

	// verify if the trace files are created
	done := verify(dst1)
	require.Equal(t, true, done)
	done = verify(dst2)
	require.Equal(t, true, done)

	// delete the traces
	deleteTraces(output)
}

// traceBlock calls the cli command to trace a block
func traceBlock(port string, number int64, output string) int {
	ui := cli.NewMockUi()
	command := &DebugBlockCommand{
		Meta2: &Meta2{
			UI:   ui,
			addr: "127.0.0.1:" + port,
		},
	}

	// run trace (by explicitly passing the output directory and grpc address)
	return command.Run([]string{strconv.FormatInt(number, 10), "--output", output, "--address", command.Meta2.addr})
}

// verify checks if the trace file is created at the destination
// directory or not
func verify(dst string) bool {
	dst = path.Join(currentDir, dst)
	if file, err := os.Stat(dst); err == nil {
		// check if the file has content
		if file.Size() > 0 {
			return true
		}
	}

	return false
}

// deleteTraces removes the traces created during the test
func deleteTraces(dst string) {
	dst = path.Join(currentDir, dst)
	os.RemoveAll(dst)
}
