package tracers

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFirehose_BlockPrintsToFirehose(t *testing.T) {
	f := NewFirehose(&FirehoseConfig{
		ApplyBackwardCompatibility: ptr(false),
		private: &privateFirehoseConfig{
			FlushToTestBuffer: true,
		},
	})

	f.OnBlockchainInit(params.AllEthashProtocolChanges)

	f.OnBlockStart(blockEvent(123))
	blockHash := hex.EncodeToString(f.block.Hash) // Store the block hash before it gets reset
	f.onTxStart(txEvent(), hex2Hash("ABCD"), from, to)
	f.OnCallEnter(0, byte(vm.CALL), from, to, nil, 0, nil)
	f.OnBalanceChange(from, b(100), b(50), 0)
	f.OnCallExit(0, nil, 0, nil, false)
	f.OnTxEnd(txReceiptEvent(0), nil)
	f.OnBlockEnd(nil)

	output := f.InternalTestingBuffer().String()

	require.Contains(t, output, "FIRE BLOCK", "expected FIRE BLOCK output not found")
	require.Contains(t, output, "123", "expected block number not found in output")
	require.Contains(t, output, blockHash, "expected block hash not found in output")
}
