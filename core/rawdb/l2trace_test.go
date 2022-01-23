package rawdb

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rlp"
)

func TestBlockEvmTracesStorage(t *testing.T) {
	db := NewMemoryDatabase()

	data1 := []byte(`{
  "gas": 1,
  "failed": false,
  "returnValue": "",
  "structLogs": [
    {
      "pc": 0,
      "op": "PUSH1",
      "gas": 1000000,
      "gasCost": 3,
      "depth": 1,
      "stack": [],
      "memory": []
    },
    {
      "pc": 2,
      "op": "SLOAD",
      "gas": 999997,
      "gasCost": 2100,
      "depth": 1,
      "stack": [
        "0x1"
      ],
      "memory": [],
      "storage": {
        "0000000000000000000000000000000000000000000000000000000000000001": "0000000000000000000000000000000000000000000000000000000000000000"
      }
    },
    {
      "pc": 3,
      "op": "POP",
      "gas": 997897,
      "gasCost": 2,
      "depth": 1,
      "stack": [
        "0x0"
      ],
      "memory": []
    },
    {
      "pc": 4,
      "op": "PUSH1",
      "gas": 997895,
      "gasCost": 3,
      "depth": 1,
      "stack": [],
      "memory": []
    },
    {
      "pc": 6,
      "op": "PUSH1",
      "gas": 997892,
      "gasCost": 3,
      "depth": 1,
      "stack": [
        "0x11"
      ],
      "memory": []
    },
    {
      "pc": 8,
      "op": "SSTORE",
      "gas": 997889,
      "gasCost": 20000,
      "depth": 1,
      "stack": [
        "0x11",
        "0x1"
      ],
      "memory": [],
      "storage": {
        "0000000000000000000000000000000000000000000000000000000000000001": "0000000000000000000000000000000000000000000000000000000000000011"
      }
    },
    {
      "pc": 9,
      "op": "PUSH1",
      "gas": 977889,
      "gasCost": 3,
      "depth": 1,
      "stack": [],
      "memory": []
    },
    {
      "pc": 11,
      "op": "PUSH1",
      "gas": 977886,
      "gasCost": 3,
      "depth": 1,
      "stack": [
        "0x11"
      ],
      "memory": []
    },
    {
      "pc": 13,
      "op": "SSTORE",
      "gas": 977883,
      "gasCost": 22100,
      "depth": 1,
      "stack": [
        "0x11",
        "0x2"
      ],
      "memory": [],
      "storage": {
        "0000000000000000000000000000000000000000000000000000000000000001": "0000000000000000000000000000000000000000000000000000000000000011",
        "0000000000000000000000000000000000000000000000000000000000000002": "0000000000000000000000000000000000000000000000000000000000000011"
      }
    },
    {
      "pc": 14,
      "op": "PUSH1",
      "gas": 955783,
      "gasCost": 3,
      "depth": 1,
      "stack": [],
      "memory": []
    },
    {
      "pc": 16,
      "op": "PUSH1",
      "gas": 955780,
      "gasCost": 3,
      "depth": 1,
      "stack": [
        "0x11"
      ],
      "memory": []
    },
    {
      "pc": 18,
      "op": "SSTORE",
      "gas": 955777,
      "gasCost": 100,
      "depth": 1,
      "stack": [
        "0x11",
        "0x2"
      ],
      "memory": [],
      "storage": {
        "0000000000000000000000000000000000000000000000000000000000000001": "0000000000000000000000000000000000000000000000000000000000000011",
        "0000000000000000000000000000000000000000000000000000000000000002": "0000000000000000000000000000000000000000000000000000000000000011"
      }
    },
    {
      "pc": 19,
      "op": "PUSH1",
      "gas": 955677,
      "gasCost": 3,
      "depth": 1,
      "stack": [],
      "memory": []
    },
    {
      "pc": 21,
      "op": "SLOAD",
      "gas": 955674,
      "gasCost": 100,
      "depth": 1,
      "stack": [
        "0x2"
      ],
      "memory": [],
      "storage": {
        "0000000000000000000000000000000000000000000000000000000000000001": "0000000000000000000000000000000000000000000000000000000000000011",
        "0000000000000000000000000000000000000000000000000000000000000002": "0000000000000000000000000000000000000000000000000000000000000011"
      }
    },
    {
      "pc": 22,
      "op": "PUSH1",
      "gas": 955574,
      "gasCost": 3,
      "depth": 1,
      "stack": [
        "0x11"
      ],
      "memory": []
    },
    {
      "pc": 24,
      "op": "SLOAD",
      "gas": 955571,
      "gasCost": 100,
      "depth": 1,
      "stack": [
        "0x11",
        "0x1"
      ],
      "memory": [],
      "storage": {
        "0000000000000000000000000000000000000000000000000000000000000001": "0000000000000000000000000000000000000000000000000000000000000011",
        "0000000000000000000000000000000000000000000000000000000000000002": "0000000000000000000000000000000000000000000000000000000000000011"
      }
    },
    {
      "pc": 25,
      "op": "STOP",
      "gas": 955471,
      "gasCost": 0,
      "depth": 1,
      "stack": [
        "0x11",
        "0x11"
      ],
      "memory": []
    }
  ]
}`)
	evmTrace1 := &types.ExecutionResult{ReturnValue: "0xaaa"}
	if err := json.Unmarshal(data1, evmTrace1); err != nil {
		t.Fatalf(err.Error())
	}

	data2 := []byte(`{
  "gas": 1,
  "failed": false,
  "returnValue": "000000000000000000000000000000000000000000000000000000000000000a",
  "structLogs": [
    {
      "pc": 0,
      "op": "PUSH1",
      "gas": 1000000,
      "gasCost": 3,
      "depth": 1,
      "stack": [],
      "memory": []
    },
    {
      "pc": 2,
      "op": "PUSH1",
      "gas": 999997,
      "gasCost": 3,
      "depth": 1,
      "stack": [
        "0xa"
      ],
      "memory": []
    },
    {
      "pc": 4,
      "op": "MSTORE",
      "gas": 999994,
      "gasCost": 6,
      "depth": 1,
      "stack": [
        "0xa",
        "0x0"
      ],
      "memory": [
        "0000000000000000000000000000000000000000000000000000000000000000"
      ]
    },
    {
      "pc": 5,
      "op": "PUSH1",
      "gas": 999988,
      "gasCost": 3,
      "depth": 1,
      "stack": [],
      "memory": [
        "000000000000000000000000000000000000000000000000000000000000000a"
      ]
    },
    {
      "pc": 7,
      "op": "PUSH1",
      "gas": 999985,
      "gasCost": 3,
      "depth": 1,
      "stack": [
        "0x20"
      ],
      "memory": [
        "000000000000000000000000000000000000000000000000000000000000000a"
      ]
    },
    {
      "pc": 9,
      "op": "RETURN",
      "gas": 999982,
      "gasCost": 0,
      "depth": 1,
      "stack": [
        "0x20",
        "0x0"
      ],
      "memory": [
        "000000000000000000000000000000000000000000000000000000000000000a"
      ]
    }
  ]
}`)
	evmTrace2 := &types.ExecutionResult{ReturnValue: "0xbbb"}
	if err := json.Unmarshal(data2, evmTrace2); err != nil {
		t.Fatalf(err.Error())
	}

	evmTraces := []*types.ExecutionResult{evmTrace1, evmTrace2}
	hash := common.BytesToHash([]byte{0x03, 0x04})
	// Insert the blockResult into the database and check presence.
	WriteBlockResult(db, hash, &types.BlockResult{ExecutionResults: evmTraces})
	// Read blockResult from db.
	if blockResult := ReadBlockResult(db, hash); len(blockResult.ExecutionResults) == 0 {
		t.Fatalf("No evmTraces returned")
	} else {
		if err := checkEvmTracesRLP(blockResult.ExecutionResults, evmTraces); err != nil {
			t.Fatalf(err.Error())
		}
	}
	// Delete blockResult by blockHash.
	DeleteBlockResult(db, hash)
	if blockResult := ReadBlockResult(db, hash); blockResult != nil && len(blockResult.ExecutionResults) != 0 {
		t.Fatalf("The evmTrace list should be empty.")
	}
}

func checkEvmTracesRLP(have, want []*types.ExecutionResult) error {
	if len(have) != len(want) {
		return fmt.Errorf("evmTraces sizes mismatch: have: %d, want: %d", len(have), len(want))
	}
	for i := 0; i < len(want); i++ {
		rlpHave, err := rlp.EncodeToBytes(have[i])
		if err != nil {
			return err
		}
		rlpWant, err := rlp.EncodeToBytes(want[i])
		if err != nil {
			return err
		}
		if !bytes.Equal(rlpHave, rlpWant) {
			return fmt.Errorf("evmTrace #%d: evmTrace mismatch: have %s, want %s", i, hex.EncodeToString(rlpHave), hex.EncodeToString(rlpWant))
		}
	}
	return nil
}
