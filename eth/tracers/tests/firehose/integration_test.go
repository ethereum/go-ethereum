package firehose_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	pbeth "github.com/ethereum/go-ethereum/pb/sf/ethereum/type/v2"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestFirehoseIntegrationTest(t *testing.T) {
	context := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(uint64(1)),
		Time:        1,
		Difficulty:  big.NewInt(2),
		GasLimit:    uint64(1000000),
		BaseFee:     big.NewInt(8),
	}

	tracer, err := tracers.NewFirehoseFromRawJSON(json.RawMessage(`{
		"applyBackwardsCompatibility": true,
		"_private": {
			"flushToTestBuffer": true,
			"ignoreGenesisBlock": true
		}
	}`))
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create firehose tracer: %w", err))
	}
	hooks := tracers.NewTracingHooksFromFirehose(tracer)

	genesis, blockchain := newCanonical(t, types.GenesisAlloc{}, context, hooks)

	block := types.NewBlock(&types.Header{
		ParentHash:       genesis.ToBlock().Hash(),
		Number:           context.BlockNumber,
		Difficulty:       context.Difficulty,
		Coinbase:         context.Coinbase,
		Time:             context.Time,
		GasLimit:         context.GasLimit,
		BaseFee:          context.BaseFee,
		ParentBeaconRoot: ptr(common.Hash{}),
	}, nil, nil, nil, trie.NewStackTrie(nil))

	blockchain.SetBlockValidatorAndProcessorForTesting(
		ignoreValidateStateValidator{core.NewBlockValidator(genesis.Config, blockchain, blockchain.Engine())},
		core.NewStateProcessor(genesis.Config, blockchain, blockchain.Engine()),
	)

	n, err := blockchain.InsertChain(types.Blocks{block})
	require.NoError(t, err)
	require.Equal(t, 1, n)

	genesisLine, blockLines, unknownLines := readTracerFirehoseLines(t, tracer)
	require.Len(t, unknownLines, 0, "Lines:\n%s", strings.Join(slicesMap(unknownLines, func(l unknwonLine) string { return "- '" + string(l) + "'" }), "\n"))
	require.NotNil(t, genesisLine)
	blockLines.assertEquals(t,
		firehoseBlockLineParams{"1", "8e6ee4b1054d94df1d8a51fb983447dc2e27a854590c3ac0061f994284be8150", "0", "845bad515694a416bab4b8d44e22cf97a8c894a8502110ab807883940e185ce0", "0", "1000000000"},
	)
}

type firehoseInitLine struct {
	ProtocolVersion string
	NodeName        string
	NodeVersion     string
}

type firehoseBlockLines []firehoseBlockLine

func (lines firehoseBlockLines) assertEquals(t *testing.T, expected ...firehoseBlockLineParams) {
	actualParams := slicesMap(lines, func(l firehoseBlockLine) firehoseBlockLineParams { return l.Params })
	require.Equal(t, expected, actualParams, "Actual lines block params do not match expected lines block params")

	goldenUpdate := os.Getenv("GOLDEN_UPDATE") == "true"

	for _, line := range lines {
		goldenPath := fmt.Sprintf("testdata/%s/block.%d.golden.json", t.Name(), line.Block.Header.Number)
		if !goldenUpdate && !fileExists(t, goldenPath) {
			t.Fatalf("the golden file %q does not exist, re-run with 'GOLDEN_UPDATE=true go test ./... -run %q' to generate the intial version", goldenPath, t.Name())
		}

		content, err := protojson.MarshalOptions{Indent: "  "}.Marshal(line.Block)
		require.NoError(t, err)

		if goldenUpdate {
			require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
			require.NoError(t, os.WriteFile(goldenPath, content, 0644))
		}

		expected, err := os.ReadFile(goldenPath)
		require.NoError(t, err)

		expectedBlock := &pbeth.Block{}
		require.NoError(t, protojson.Unmarshal(expected, expectedBlock))

		if !proto.Equal(expectedBlock, line.Block) {
			assert.EqualExportedValues(t, expectedBlock, line.Block, "Run 'GOLDEN_UPDATE=true go test ./... -run %q' to update golden file", t.Name())
		}
	}
}

func fileExists(t *testing.T, path string) bool {
	t.Helper()
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}

		t.Fatal(err)
	}

	return !stat.IsDir()
}

func slicesMap[T any, U any](s []T, f func(T) U) []U {
	result := make([]U, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}

type firehoseBlockLine struct {
	// We split params and block to make it easier to compare stuff
	Params firehoseBlockLineParams
	Block  *pbeth.Block
}

type firehoseBlockLineParams struct {
	Number       string
	Hash         string
	PreviousNum  string
	PreviousHash string
	LibNum       string
	Time         string
}

type unknwonLine string

func readTracerFirehoseLines(t *testing.T, tracer *tracers.Firehose) (genesisLine *firehoseInitLine, blockLines firehoseBlockLines, unknownLines []unknwonLine) {
	t.Helper()

	lines := bytes.Split(tracer.InternalTestingBuffer().Bytes(), []byte{'\n'})
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		parts := bytes.Split(line, []byte{' '})
		if len(parts) == 0 || string(parts[0]) != "FIRE" {
			unknownLines = append(unknownLines, unknwonLine(line))
			continue
		}

		action := string(parts[1])
		fireParts := parts[2:]
		switch action {
		case "INIT":
			genesisLine = &firehoseInitLine{
				ProtocolVersion: string(fireParts[0]),
				NodeName:        string(fireParts[1]),
				NodeVersion:     string(fireParts[2]),
			}

		case "BLOCK":
			protoBytes, err := base64.StdEncoding.DecodeString(string(fireParts[6]))
			require.NoError(t, err)

			block := &pbeth.Block{}
			require.NoError(t, proto.Unmarshal(protoBytes, block))

			blockLines = append(blockLines, firehoseBlockLine{
				Params: firehoseBlockLineParams{
					Number:       string(fireParts[0]),
					Hash:         string(fireParts[1]),
					PreviousNum:  string(fireParts[2]),
					PreviousHash: string(fireParts[3]),
					LibNum:       string(fireParts[4]),
					Time:         string(fireParts[5]),
				},
				Block: block,
			})

		default:
			unknownLines = append(unknownLines, unknwonLine(line))
		}
	}

	return
}

func newCanonical(t *testing.T, alloc types.GenesisAlloc, context vm.BlockContext, tracer *tracing.Hooks) (*core.Genesis, *core.BlockChain) {
	t.Helper()

	var (
		engine  = ethash.NewFullFaker()
		genesis = &core.Genesis{
			Difficulty: new(big.Int).Sub(context.Difficulty, big.NewInt(1)),
			Timestamp:  context.Time - 1,
			Number:     new(big.Int).Sub(context.BlockNumber, big.NewInt(1)).Uint64(),
			BaseFee:    big.NewInt(params.InitialBaseFee),
			Coinbase:   context.Coinbase,
			Config:     params.AllEthashProtocolChanges,
			Alloc:      alloc,
		}
	)
	// Initialize a fresh chain with only a genesis block
	blockchain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), core.DefaultCacheConfigWithScheme(rawdb.HashScheme), genesis, nil, engine, vm.Config{
		Tracer: tracer,
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	return genesis, blockchain
}

// testHasher is the helper tool for transaction/receipt list hashing.
// The original hasher is trie, in order to get rid of import cycle,
// use the testing hasher instead.
type testHasher struct {
	hasher hash.Hash
}

// NewHasher returns a new testHasher instance.
func NewHasher() *testHasher {
	return &testHasher{hasher: sha3.NewLegacyKeccak256()}
}

// Reset resets the hash state.
func (h *testHasher) Reset() {
	h.hasher.Reset()
}

// Update updates the hash state with the given key and value.
func (h *testHasher) Update(key, val []byte) error {
	h.hasher.Write(key)
	h.hasher.Write(val)
	return nil
}

// Hash returns the hash value.
func (h *testHasher) Hash() common.Hash {
	return common.BytesToHash(h.hasher.Sum(nil))
}

type ignoreValidateStateValidator struct {
	core.Validator
}

func (v ignoreValidateStateValidator) ValidateBody(block *types.Block) error {
	return v.Validator.ValidateBody(block)
}

func (v ignoreValidateStateValidator) ValidateState(block *types.Block, statedb *state.StateDB, receipts types.Receipts, usedGas uint64) error {
	return nil
}

func ptr[T any](v T) *T {
	return &v
}
