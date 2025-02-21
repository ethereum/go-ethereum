package firehose_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	pbeth "github.com/streamingfast/firehose-ethereum/types/pb/sf/ethereum/type/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type firehoseInitLine struct {
	ProtocolVersion string
	NodeName        string
	NodeVersion     string
}

type firehoseBlockLines []firehoseBlockLine

func newFirehoseTestTracer(t *testing.T) (*tracers.Firehose, *tracing.Hooks, func()) {
	t.Helper()

	tracer, err := tracers.NewFirehoseFromRawJSON([]byte(`{
		"applyBackwardsCompatibility": true,
		"_private": {
			"flushToTestBuffer": true,
			"ignoreGenesisBlock": true
		}
	}`))
	require.NoError(t, err)

	hooks := tracers.NewTracingHooksFromFirehose(tracer)

	return tracer, hooks, func() {
		if hooks.OnClose != nil {
			hooks.OnClose()
		}
	}
}

func (lines firehoseBlockLines) assertEquals(t *testing.T, goldenDir string, expected ...firehoseBlockLineParams) {
	actualParams := slicesMap(lines, func(l firehoseBlockLine) firehoseBlockLineParams { return l.Params })
	require.Equal(t, expected, actualParams, "Actual lines block params do not match expected lines block params")

	lines.assertOnlyBlockEquals(t, goldenDir, len(expected))
}

func (lines firehoseBlockLines) assertOnlyBlockEquals(t *testing.T, goldenDir string, expectedBlockCount int) {
	t.Helper()

	require.Len(t, lines, expectedBlockCount, "Expected %d blocks, got %d", expectedBlockCount, len(lines))
	goldenUpdate := os.Getenv("GOLDEN_UPDATE") == "true"

	for _, line := range lines {
		goldenPath := filepath.Join(goldenDir, fmt.Sprintf("block.%d.golden.json", line.Block.Header.Number))
		if !goldenUpdate && !fileExists(t, goldenPath) {
			t.Fatalf("the golden file %q does not exist, re-run with 'GOLDEN_UPDATE=true go test ./... -run %q' to generate the intial version", goldenPath, t.Name())
		}

		unnormalizedContent, err := protojson.MarshalOptions{Indent: "  "}.Marshal(line.Block)
		require.NoError(t, err)

		if goldenUpdate {
			content := normalizedJSON(t, unnormalizedContent)

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

type unknownLine string

// assertBlockEquals reads the tracer output and compares it to the golden files in the given directory.
func assertBlockEquals(t *testing.T, tracer *tracers.Firehose, goldenDir string, expectedBlockCount int) {
	t.Helper()

	genesisLine, blockLines, unknownLines := readTracerFirehoseLines(t, tracer)
	require.Len(t, unknownLines, 0, "Lines:\n%s", strings.Join(slicesMap(unknownLines, func(l unknownLine) string { return "- '" + string(l) + "'" }), "\n"))
	require.NotNil(t, genesisLine)
	blockLines.assertOnlyBlockEquals(t, goldenDir, expectedBlockCount)
}

func readTracerFirehoseLines(t *testing.T, tracer *tracers.Firehose) (genesisLine *firehoseInitLine, blockLines firehoseBlockLines, unknownLines []unknownLine) {
	t.Helper()

	lines := bytes.Split(tracer.InternalTestingBuffer().Bytes(), []byte{'\n'})
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		parts := bytes.Split(line, []byte{' '})
		if len(parts) == 0 || string(parts[0]) != "FIRE" {
			unknownLines = append(unknownLines, unknownLine(line))
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
			unknownLines = append(unknownLines, unknownLine(line))
		}
	}

	return
}

func ptr[T any](v T) *T {
	return &v
}

func newGwei(n int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(n), big.NewInt(params.GWei))
}

func normalizedJSON(t *testing.T, data []byte) []byte {
	t.Helper()

	var obj map[string]any
	require.NoError(t, json.Unmarshal(data, &obj))

	normalized, err := json.MarshalIndent(obj, "", "  ")
	require.NoError(t, err)

	return normalized
}
