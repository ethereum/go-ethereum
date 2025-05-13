package tracers

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestFirehose_BlockPrintsToFirehose_SingleBlock(t *testing.T) {

	f := NewFirehose(&FirehoseConfig{
		ConcurrentBlockFlushing:    true,
		ApplyBackwardCompatibility: ptr(false),
		private: &privateFirehoseConfig{
			FlushToTestBuffer: true,
		},
	})

	f.OnBlockchainInit(params.AllEthashProtocolChanges)

	blockNumbers := []uint64{123, 124, 125}

	for i, blockNum := range blockNumbers {
		f.OnBlockStart(blockEvent(blockNum))

		f.onTxStart(txEvent(), hex2Hash(fmt.Sprintf("ABCD%d", i)), from, to)
		f.OnCallEnter(0, byte(vm.CALL), from, to, nil, 0, nil)
		f.OnBalanceChange(from, b(100), b(50), 0)
		f.OnCallExit(0, nil, 0, nil, false)
		f.OnTxEnd(txReceiptEvent(0), nil)

		f.OnBlockEnd(nil)
	}

	f.OnClose()

	output := f.InternalTestingBuffer().String()

	outNumber := make([]string, 0)
	for i, line := range strings.Split(output, "\n") {
		if i == 0 {
			require.Equal(t, "FIRE INIT 3.0 geth 1.15.10", line)
			continue
		}

		fields := strings.SplitN(line, " ", 4)
		if len(fields) >= 3 {
			require.Equal(t, "FIRE", fields[0])
			require.Equal(t, "BLOCK", fields[1])
			outNumber = append(outNumber, fields[2])
		}
	}

	require.Equal(t, []string{"123", "124", "125"}, outNumber)
}

func TestFirehose_BlocksPrintToFirehose_MultipleBlocksInOrder(t *testing.T) {

	const blockCount = 10
	const baseBlockNum = 1000

	f := NewFirehose(&FirehoseConfig{
		ConcurrentBlockFlushing:    true,
		ApplyBackwardCompatibility: ptr(false),
		private: &privateFirehoseConfig{
			FlushToTestBuffer: true,
		},
	})

	f.OnBlockchainInit(params.AllEthashProtocolChanges)

	blockHashes := make(map[uint64]string, blockCount)

	for i := 0; i < blockCount; i++ {
		blockNum := uint64(baseBlockNum + i)

		f.OnBlockStart(blockEvent(blockNum))
		blockHashes[blockNum] = hex.EncodeToString(f.block.Hash) // Store hash before block reset

		f.onTxStart(txEvent(), hex2Hash(fmt.Sprintf("TX%d", i)), from, to)
		f.OnCallEnter(0, byte(vm.CALL), from, to, nil, 0, nil)
		f.OnBalanceChange(from, b(100), b(50), 0)
		f.OnCallExit(0, nil, 0, nil, false)
		f.OnTxEnd(txReceiptEvent(0), nil)

		f.OnBlockEnd(nil)
	}

	time.Sleep(5 * time.Second)

	f.OnClose()

	output := f.InternalTestingBuffer().String()
	extractedBlocks := extractBlocksFromOutput(t, output)

	// Verify block count
	require.Equal(t, blockCount, len(extractedBlocks),
		"Expected %d blocks in output, found %d", blockCount, len(extractedBlocks))

	// Verify blocks in order
	verifyBlockSequence(t, extractedBlocks, baseBlockNum)

	// Verify block hashes
	for _, block := range extractedBlocks {
		expectedHash, exists := blockHashes[block.number]
		require.True(t, exists, "Block %d not found in tracked blocks", block.number)
		require.Equal(t, expectedHash, block.hash,
			"Hash mismatch for block %d", block.number)
	}
}

type extractedBlock struct {
	number uint64
	hash   string
}

func extractBlocksFromOutput(t *testing.T, output string) []extractedBlock {
	t.Helper()

	// Regex to extract the block number and hash from the FIRE BLOCK line
	blockInfoRegex := regexp.MustCompile(`FIRE BLOCK (\d+) ([0-9a-fA-F]+)`)

	lines := strings.Split(output, "\n")
	var blocks []extractedBlock

	for _, line := range lines {
		if strings.HasPrefix(line, "FIRE BLOCK") {
			matches := blockInfoRegex.FindStringSubmatch(line)
			if len(matches) == 3 {
				blockNumStr := matches[1]
				blockHash := matches[2]

				blockNum, err := strconv.ParseUint(blockNumStr, 10, 64)
				require.NoError(t, err, "failed to parse block number: %s", blockNumStr)

				blocks = append(blocks, extractedBlock{
					number: blockNum,
					hash:   blockHash,
				})
			}
		}
	}

	return blocks
}

func verifyBlockSequence(t *testing.T, blocks []extractedBlock, baseBlockNum uint64) {
	t.Helper()

	// First block should be the base block number
	require.Equal(t, baseBlockNum, blocks[0].number,
		"First block should be %d, got %d", baseBlockNum, blocks[0].number)

	// Last block should be base + count - 1
	expectedLast := baseBlockNum + uint64(len(blocks)) - 1
	require.Equal(t, expectedLast, blocks[len(blocks)-1].number,
		"Last block should be %d, got %d", expectedLast, blocks[len(blocks)-1].number)

	// Verify sequence
	for i := 0; i < len(blocks)-1; i++ {
		current := blocks[i].number
		next := blocks[i+1].number
		require.Equal(t, current+1, next,
			"Blocks out of order at position %d: %d followed by %d", i, current, next)
	}
}
