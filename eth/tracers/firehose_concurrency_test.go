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

	blockNumbers := []uint64{0}

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

	lines := strings.Split(strings.TrimSpace(f.InternalTestingBuffer().String()), "\n")
	require.Len(t, lines, 2)

	fieldsInit := strings.SplitN(lines[0], " ", 3)
	require.Equal(t, "FIRE", fieldsInit[0])
	require.Equal(t, "INIT", fieldsInit[1])
	require.Contains(t, fieldsInit[2], "geth")

	fields := strings.SplitN(lines[1], " ", 4)
	require.GreaterOrEqual(t, len(fields), 3)
	require.Equal(t, "FIRE", fields[0])
	require.Equal(t, "BLOCK", fields[1])
	require.Equal(t, "0", fields[2])
}

func TestFirehose_BlocksPrintToFirehose_MultipleBlocksInOrder(t *testing.T) {

	const blockCount = 100
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

	f.OnClose()

	output := f.InternalTestingBuffer().String()
	extractedBlocks := extractBlocksFromOutput(t, output)

	// Verify block count
	require.Equal(t, blockCount, len(extractedBlocks),
		"Expected %d blocks in output, found %d", blockCount, len(extractedBlocks))

	// Verify blocks in order
	for i, block := range extractedBlocks {
		require.Equal(t, baseBlockNum+uint64(i), block.number, "Blocks out of order at position %d", i)
	}

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
