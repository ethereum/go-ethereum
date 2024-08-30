package live

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/ethereum/go-ethereum/tests"
)

func TestSupplyTracerBlockchain(t *testing.T) {
	dirPath := "supply"
	files, err := os.ReadDir(filepath.Join("testdata", dirPath))
	if err != nil {
		t.Fatalf("failed to retrieve tracer test suite: %v", err)
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		file := file // capture range variable
		var testcases map[string]*tests.BlockTest
		var blob []byte
		// Call tracer test found, read if from disk
		if blob, err = os.ReadFile(filepath.Join("testdata", dirPath, file.Name())); err != nil {
			t.Fatalf("failed to read testcase: %v", err)
		}
		if err := json.Unmarshal(blob, &testcases); err != nil {
			t.Fatalf("failed to parse testcase: %v", err)
		}
		for testname, test := range testcases {
			t.Run(fmt.Sprintf("%s/%s", camel(strings.TrimSuffix(file.Name(), ".json")), testname), func(t *testing.T) {
				t.Parallel()

				traceOutputPath := filepath.ToSlash(t.TempDir())
				traceOutputFilename := path.Join(traceOutputPath, "supply.jsonl")
				// Load supply tracer
				tracer, err := newSupply(json.RawMessage(fmt.Sprintf(`{"path":"%s"}`, traceOutputPath)))
				if err != nil {
					t.Fatalf("failed to create call tracer: %v", err)
				}
				if err := test.Run(false, "path", false, tracer, nil); err != nil {
					t.Errorf("failed to run test: %v\n", err)
				}
				// Check and compare the results
				file, err := os.OpenFile(traceOutputFilename, os.O_RDONLY, 0666)
				if err != nil {
					t.Fatalf("failed to open output file: %v", err)
				}
				defer file.Close()
				var output []supplyInfo
				scanner := bufio.NewScanner(file)

				for scanner.Scan() {
					blockBytes := scanner.Bytes()

					var info supplyInfo
					if err := json.Unmarshal(blockBytes, &info); err != nil {
						t.Fatalf("failed to unmarshal result: %v", err)
					}

					output = append(output, info)
				}
				fmt.Printf("output: %v\n", output)
			})
		}
	}
}

// camel converts a snake cased input string into a camel cased output.
func camel(str string) string {
	pieces := strings.Split(str, "_")
	for i := 1; i < len(pieces); i++ {
		pieces[i] = string(unicode.ToUpper(rune(pieces[i][0]))) + pieces[i][1:]
	}
	return strings.Join(pieces, "")
}
