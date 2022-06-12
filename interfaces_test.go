// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethereum

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

var blockHash = common.HexToHash(
	"0xeb94bb7d78b73657a9d7a99792413f50c0a45c51fc62bdcb08a53f18e9a2b4eb",
)

func TestFilterQuery_MarshalJSON(t *testing.T) {
	var blockHashErr = fmt.Errorf("cannot specify both BlockHash and FromBlock/ToBlock")
	addresses := []common.Address{
		common.HexToAddress("0xD36722ADeC3EdCB29c8e7b5a47f352D701393462"),
	}

	for _, testCase := range []struct {
		name   string
		input  FilterQuery
		output string
		err    error
	}{
		{
			name: "with all the fields except blockHash set",
			input: FilterQuery{
				FromBlock: big.NewInt(1234567890),
				ToBlock:   big.NewInt(2345678901),
				Addresses: []common.Address{
					common.HexToAddress("0xb26f2b342aab24bcf63ea218c6a9274d30ab9a15"),
					common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
				},
				Topics: [][]common.Hash{
					{},
					{common.HexToHash("0xd783efa4d392943503f28438ad5830b2d5964696ffc285f338585e9fe0a37a05")},
					{
						common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
						common.HexToHash("0x77d14e10470b5850332524f8cd6f69ad21f070ce92dca33ab2858300242ef2f1"),
					},
				},
			},
			output: `{` +
				`"address":["0xb26f2b342aab24bcf63ea218c6a9274d30ab9a15","0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"],` +
				`"fromBlock":"0x499602d2",` +
				`"toBlock":"0x8bd03835",` +
				`"topics":[` +
				`[],` +
				`["0xd783efa4d392943503f28438ad5830b2d5964696ffc285f338585e9fe0a37a05"],` +
				`["0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","0x77d14e10470b5850332524f8cd6f69ad21f070ce92dca33ab2858300242ef2f1"]` +
				`]` +
				`}`,
		},
		{
			name: "without BlockHash",
			input: FilterQuery{
				Addresses: addresses,
				FromBlock: big.NewInt(1),
				ToBlock:   big.NewInt(2),
				Topics:    [][]common.Hash{},
			},
			output: `{` +
				`"address":["0xd36722adec3edcb29c8e7b5a47f352d701393462"],` +
				`"fromBlock":"0x1",` +
				`"toBlock":"0x2",` +
				`"topics":[]` +
				`}`,
		},
		{
			name: "with nil fromBlock and nil toBlock",
			input: FilterQuery{
				Addresses: addresses,
				Topics:    [][]common.Hash{},
			},
			output: `{` +
				`"address":["0xd36722adec3edcb29c8e7b5a47f352d701393462"],` +
				`"topics":[]` +
				`}`,
		},
		{
			name: "with fromBlock=-1 and negative toBlock=-2",
			input: FilterQuery{
				Addresses: addresses,
				FromBlock: big.NewInt(-1),
				ToBlock:   big.NewInt(-2),
				Topics:    [][]common.Hash{},
			},
			output: `{` +
				`"address":["0xd36722adec3edcb29c8e7b5a47f352d701393462"],` +
				`"fromBlock":"latest",` +
				`"toBlock":"pending",` +
				`"topics":[]` +
				`}`,
		},
		{
			name: "with fromBlock=-2 and negative toBlock=-1",
			input: FilterQuery{
				Addresses: addresses,
				FromBlock: big.NewInt(-2),
				ToBlock:   big.NewInt(-1),
				Topics:    [][]common.Hash{},
			},
			output: `{` +
				`"address":["0xd36722adec3edcb29c8e7b5a47f352d701393462"],` +
				`"fromBlock":"pending",` +
				`"toBlock":"latest",` +
				`"topics":[]` +
				`}`,
		},
		{
			name: "with blockhash",
			input: FilterQuery{
				Addresses: addresses,
				BlockHash: &blockHash,
				Topics:    [][]common.Hash{},
			},
			output: `{` +
				`"address":["0xd36722adec3edcb29c8e7b5a47f352d701393462"],` +
				`"blockHash":"0xeb94bb7d78b73657a9d7a99792413f50c0a45c51fc62bdcb08a53f18e9a2b4eb",` +
				`"topics":[]` +
				`}`,
		},
		{
			name: "with blockhash and from block",
			input: FilterQuery{
				Addresses: addresses,
				BlockHash: &blockHash,
				FromBlock: big.NewInt(1),
				Topics:    [][]common.Hash{},
			},
			err: blockHashErr,
		},
		{
			name: "with blockhash and to block",
			input: FilterQuery{
				Addresses: addresses,
				BlockHash: &blockHash,
				ToBlock:   big.NewInt(1),
				Topics:    [][]common.Hash{},
			},
			err: blockHashErr,
		},
		{
			name: "with blockhash and both from / to block",
			input: FilterQuery{
				Addresses: addresses,
				BlockHash: &blockHash,
				FromBlock: big.NewInt(1),
				ToBlock:   big.NewInt(2),
				Topics:    [][]common.Hash{},
			},
			err: blockHashErr,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			output, err := json.Marshal(testCase.input)
			if (testCase.err == nil) != (err == nil) {
				t.Fatalf("expected error %v but got %v", testCase.err, err)
			}
			if testCase.err != nil {
				if testCase.err.Error() != errors.Unwrap(err).Error() {
					t.Fatalf("expected error %v but got %v", testCase.err, err)
				}
				assert.Nil(t, output)
			} else {
				assert.Equal(t, testCase.output, string(output))
			}
		})
	}
}

func TestFilterQuery_UnmarshalJSON(t *testing.T) {
	var (
		fromBlock    rpc.BlockNumber = 0x123435
		toBlock      rpc.BlockNumber = 0xabcdef
		address0                     = common.HexToAddress("70c87d191324e6712a591f304b4eedef6ad9bb9d")
		address1                     = common.HexToAddress("9b2055d370f73ec7d8a03e965129118dc8f5bf83")
		topic0                       = common.HexToHash("3ac225168df54212a25c1c01fd35bebfea408fdac2e31ddd6f80a4bbf9a5f1ca")
		topic1                       = common.HexToHash("9084a792d2f8b16a62b882fd56f7860c07bf5fa91dd8a2ae7e809e5180fef0b3")
		topic2                       = common.HexToHash("6ccae1c4af4152f460ff510e573399795dfab5dcf1fa60d1f33ac8fdc1e480ce")
		blockHashErr                 = fmt.Errorf("cannot specify both BlockHash and FromBlock/ToBlock, choose one or the other")
	)

	t.Run("default values", func(t *testing.T) {
		var decoded FilterQuery
		if err := json.Unmarshal([]byte("{}"), &decoded); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, big.NewInt(rpc.EarliestBlockNumber.Int64()), decoded.FromBlock)
		assert.Equal(t, big.NewInt(rpc.LatestBlockNumber.Int64()), decoded.ToBlock)
		if len(decoded.Addresses) != 0 {
			t.Errorf("expected 0 addresses, got %d", len(decoded.Addresses))
		}
		if len(decoded.Topics) != 0 {
			t.Errorf("expected 0 topics, got %d topics", len(decoded.Topics))
		}
	})

	t.Run("with blockHash", func(t *testing.T) {
		var decoded FilterQuery
		if err := json.Unmarshal([]byte(fmt.Sprintf(`{"blockHash":"%s"}`, blockHash.String())), &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.BlockHash == nil {
			t.Fatalf("expected BlockHash to be set, but got nil")
		}
		assert.Equal(t, blockHash, *decoded.BlockHash)
		assert.Nil(t, decoded.FromBlock)
		assert.Nil(t, decoded.ToBlock)
	})

	t.Run("blockHash and fromBlock", func(t *testing.T) {
		var decoded FilterQuery
		err := json.Unmarshal([]byte(fmt.Sprintf(`{"blockHash":"%s","fromBlock":"0x0"}`, blockHash.String())), &decoded)
		assert.Equal(t, blockHashErr, err)
	})

	t.Run("blockHash and toBlock", func(t *testing.T) {
		var decoded FilterQuery
		err := json.Unmarshal([]byte(fmt.Sprintf(`{"blockHash":"%s","toBlock":"0x0"}`, blockHash.String())), &decoded)
		assert.Equal(t, blockHashErr, err)
	})

	t.Run("from, to block number", func(t *testing.T) {
		var decoded FilterQuery
		vector := fmt.Sprintf(`{"fromBlock":"0x%x","toBlock":"0x%x"}`, fromBlock, toBlock)
		if err := json.Unmarshal([]byte(vector), &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.FromBlock.Int64() != fromBlock.Int64() {
			t.Fatalf("expected FromBlock %d, got %d", fromBlock, decoded.FromBlock)
		}
		if decoded.ToBlock.Int64() != toBlock.Int64() {
			t.Fatalf("expected ToBlock %d, got %d", toBlock, decoded.ToBlock)
		}
	})

	t.Run("single address", func(t *testing.T) {
		var decoded FilterQuery
		vector := fmt.Sprintf(`{"address": "%s"}`, address0.Hex())
		if err := json.Unmarshal([]byte(vector), &decoded); err != nil {
			t.Fatal(err)
		}
		if len(decoded.Addresses) != 1 {
			t.Fatalf("expected 1 address, got %d address(es)", len(decoded.Addresses))
		}
		if decoded.Addresses[0] != address0 {
			t.Fatalf("expected address %x, got %x", address0, decoded.Addresses[0])
		}
	})

	t.Run("multiple addresses", func(t *testing.T) {
		var decoded FilterQuery
		vector := fmt.Sprintf(`{"address": ["%s", "%s"]}`, address0.Hex(), address1.Hex())
		if err := json.Unmarshal([]byte(vector), &decoded); err != nil {
			t.Fatal(err)
		}
		if len(decoded.Addresses) != 2 {
			t.Fatalf("expected 2 addresses, got %d address(es)", len(decoded.Addresses))
		}
		if decoded.Addresses[0] != address0 {
			t.Fatalf("expected address %x, got %x", address0, decoded.Addresses[0])
		}
		if decoded.Addresses[1] != address1 {
			t.Fatalf("expected address %x, got %x", address1, decoded.Addresses[1])
		}
	})

	t.Run("invalid addresses", func(t *testing.T) {
		var decoded FilterQuery
		vector := `{"address": 123}`
		err := json.Unmarshal([]byte(vector), &decoded)
		assert.Equal(t, fmt.Errorf("invalid addresses in query"), err)
	})

	t.Run("invalid address inside", func(t *testing.T) {
		var decoded FilterQuery
		vector := `{"address": [123]}`
		err := json.Unmarshal([]byte(vector), &decoded)
		assert.Equal(t, fmt.Errorf("non-string address at index 0"), err)
	})

	t.Run("address is too short", func(t *testing.T) {
		var decoded FilterQuery
		vector := `{"address": "0x12"}`
		err := json.Unmarshal([]byte(vector), &decoded)
		assert.Equal(t, fmt.Errorf("invalid address: hex has invalid length 1 after decoding; expected 20 for address"), err)
	})

	t.Run("address is too long", func(t *testing.T) {
		var decoded FilterQuery
		vector := fmt.Sprintf(`{"address": ["%s12"]}`, address0.Hex())
		err := json.Unmarshal([]byte(vector), &decoded)
		assert.Equal(t, fmt.Errorf("invalid address at index 0: hex has invalid length 21 after decoding; expected 20 for address"), err)
	})

	t.Run("address has odd length", func(t *testing.T) {
		var decoded FilterQuery
		vector := fmt.Sprintf(`{"address": "%s1"}`, address0.Hex())
		err := json.Unmarshal([]byte(vector), &decoded)
		assert.Equal(t, fmt.Errorf("invalid address: hex string of odd length"), err)
	})

	t.Run("single topic", func(t *testing.T) {
		var decoded FilterQuery
		vector := fmt.Sprintf(`{"topics": ["%s"]}`, topic0.Hex())
		if err := json.Unmarshal([]byte(vector), &decoded); err != nil {
			t.Fatal(err)
		}
		if len(decoded.Topics) != 1 {
			t.Fatalf("expected 1 topic, got %d", len(decoded.Topics))
		}
		if len(decoded.Topics[0]) != 1 {
			t.Fatalf("expected len(topics[0]) to be 1, got %d", len(decoded.Topics[0]))
		}
		if decoded.Topics[0][0] != topic0 {
			t.Fatalf("got %x, expected %x", decoded.Topics[0][0], topic0)
		}
	})

	t.Run(`multiple "AND" topics`, func(t *testing.T) {
		var decoded FilterQuery
		vector := fmt.Sprintf(`{"topics": ["%s", "%s"]}`, topic0.Hex(), topic1.Hex())
		if err := json.Unmarshal([]byte(vector), &decoded); err != nil {
			t.Fatal(err)
		}
		if len(decoded.Topics) != 2 {
			t.Fatalf("expected 2 topics, got %d", len(decoded.Topics))
		}
		if len(decoded.Topics[0]) != 1 {
			t.Fatalf("expected 1 topic, got %d", len(decoded.Topics[0]))
		}
		if decoded.Topics[0][0] != topic0 {
			t.Fatalf("got %x, expected %x", decoded.Topics[0][0], topic0)
		}
		if len(decoded.Topics[1]) != 1 {
			t.Fatalf("expected 1 topic, got %d", len(decoded.Topics[1]))
		}
		if decoded.Topics[1][0] != topic1 {
			t.Fatalf("got %x, expected %x", decoded.Topics[1][0], topic1)
		}
	})

	t.Run("optional topic", func(t *testing.T) {
		var decoded FilterQuery
		vector := fmt.Sprintf(`{"topics": ["%s", null, "%s"]}`, topic0.Hex(), topic2.Hex())
		if err := json.Unmarshal([]byte(vector), &decoded); err != nil {
			t.Fatal(err)
		}
		if len(decoded.Topics) != 3 {
			t.Fatalf("expected 3 topics, got %d", len(decoded.Topics))
		}
		if len(decoded.Topics[0]) != 1 {
			t.Fatalf("expected 1 topic, got %d", len(decoded.Topics[0]))
		}
		if decoded.Topics[0][0] != topic0 {
			t.Fatalf("got %x, expected %x", decoded.Topics[0][0], topic0)
		}
		if len(decoded.Topics[1]) != 0 {
			t.Fatalf("expected 0 topic, got %d", len(decoded.Topics[1]))
		}
		if len(decoded.Topics[2]) != 1 {
			t.Fatalf("expected 1 topic, got %d", len(decoded.Topics[2]))
		}
		if decoded.Topics[2][0] != topic2 {
			t.Fatalf("got %x, expected %x", decoded.Topics[2][0], topic2)
		}
	})

	t.Run("OR topics", func(t *testing.T) {
		var decoded FilterQuery
		vector := fmt.Sprintf(`{"topics": [["%s", "%s"], null, ["%s", null]]}`, topic0.Hex(), topic1.Hex(), topic2.Hex())
		if err := json.Unmarshal([]byte(vector), &decoded); err != nil {
			t.Fatal(err)
		}
		if len(decoded.Topics) != 3 {
			t.Fatalf("expected 3 topics, got %d topics", len(decoded.Topics))
		}
		if len(decoded.Topics[0]) != 2 {
			t.Fatalf("expected 2 topics, got %d topics", len(decoded.Topics[0]))
		}
		if decoded.Topics[0][0] != topic0 || decoded.Topics[0][1] != topic1 {
			t.Fatalf("invalid topics expected [%x,%x], got [%x,%x]",
				topic0, topic1, decoded.Topics[0][0], decoded.Topics[0][1],
			)
		}
		if len(decoded.Topics[1]) != 0 {
			t.Fatalf("expected 0 topic, got %d topics", len(decoded.Topics[1]))
		}
		if len(decoded.Topics[2]) != 0 {
			t.Fatalf("expected 0 topics, got %d topics", len(decoded.Topics[2]))
		}
	})

	t.Run("invalid topic", func(t *testing.T) {
		var decoded FilterQuery
		vector := `{"topics": [123]}`
		err := json.Unmarshal([]byte(vector), &decoded)
		assert.Equal(t, fmt.Errorf("invalid topic(s) at index 0"), err)
	})

	t.Run("invalid OR topic", func(t *testing.T) {
		var decoded FilterQuery
		vector := `{"topics": [[123]]}`
		err := json.Unmarshal([]byte(vector), &decoded)
		assert.Equal(t, fmt.Errorf("invalid topic(s) at index 0,0"), err)
	})

	t.Run("topic is too short", func(t *testing.T) {
		var decoded FilterQuery
		vector := `{"topics": [["0x12"]]}`
		err := json.Unmarshal([]byte(vector), &decoded)
		assert.Equal(t, fmt.Errorf("invalid topic at index 0,0: hex has invalid length 1 after decoding; expected 32 for topic"), err)
	})

	t.Run("topic is too long", func(t *testing.T) {
		var decoded FilterQuery
		vector := fmt.Sprintf(`{"topics": ["%s12"]}`, topic0.Hex())
		err := json.Unmarshal([]byte(vector), &decoded)
		assert.Equal(t, fmt.Errorf("invalid topic at index 0: hex has invalid length 33 after decoding; expected 32 for topic"), err)
	})

	t.Run("topic has odd length", func(t *testing.T) {
		var decoded FilterQuery
		vector := fmt.Sprintf(`{"topics": [["%s1"]]}`, topic0.Hex())
		err := json.Unmarshal([]byte(vector), &decoded)
		assert.Equal(t, fmt.Errorf("invalid topic at index 0,0: hex string of odd length"), err)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		var decoded FilterQuery
		err := json.Unmarshal([]byte("[]"), &decoded)
		assert.Error(t, err)
	})
}
