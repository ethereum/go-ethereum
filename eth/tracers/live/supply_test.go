// Copyright 2024 The go-ethereum Authors
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

package live

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/tests"
)

type blockTest struct {
	bt       *tests.BlockTest
	Expected []supplyInfo `json:"expected"`
}

func (bt *blockTest) UnmarshalJSON(data []byte) error {
	tmp := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	if err := json.Unmarshal(tmp["expected"], &bt.Expected); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &bt.bt); err != nil {
		return err
	}
	return nil
}

// The tests have been filled using the executable at
// eth/tracers/live/tests/supply_filler.go.
func TestSupplyTracerBlockchain(t *testing.T) {
	dirPath := filepath.Join("testdata", "supply")
	files, err := os.ReadDir(dirPath)
	if err != nil {
		t.Fatalf("failed to retrieve tracer test suite: %v", err)
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		var testcases map[string]*blockTest
		var blob []byte
		// Tracer test found, read if from disk
		if blob, err = os.ReadFile(filepath.Join(dirPath, file.Name())); err != nil {
			t.Fatalf("failed to read testcase: %v", err)
		}
		if err := json.Unmarshal(blob, &testcases); err != nil {
			t.Fatalf("failed to parse testcase %s: %v", file.Name(), err)
		}
		for testname, blockTest := range testcases {
			t.Run(fmt.Sprintf("%s/%s", file.Name(), testname),
				func(t *testing.T) { runBlockTest(t, blockTest) })
		}
	}
}

func runBlockTest(t *testing.T, test *blockTest) {
	t.Parallel()

	traceOutputPath := filepath.ToSlash(t.TempDir())
	traceOutputFilename := path.Join(traceOutputPath, "supply.jsonl")
	// Load supply tracer
	tracer, err := newSupply(json.RawMessage(fmt.Sprintf(`{"path":"%s"}`, traceOutputPath)))
	if err != nil {
		t.Fatalf("failed to create tracer: %v", err)
	}
	if err := test.bt.Run(false, "path", false, tracer, nil); err != nil {
		t.Errorf("failed to run test: %v\n", err)
	}
	// Check and compare the results
	file, err := os.OpenFile(traceOutputFilename, os.O_RDONLY, 0666)
	if err != nil {
		t.Fatalf("failed to open output file: %v", err)
	}
	defer file.Close()

	var (
		output  []supplyInfo
		scanner = bufio.NewScanner(file)
	)
	for scanner.Scan() {
		var info supplyInfo
		if err := json.Unmarshal(scanner.Bytes(), &info); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		output = append(output, info)
	}
	if len(output) != len(test.Expected) {
		fmt.Printf("output: %v\n", output)
		t.Fatalf("expected %d supply infos, got %d", len(test.Expected), len(output))
	}
	for i, expected := range test.Expected {
		compareAsJSON(t, expected, output[i])
	}
}

func compareAsJSON(t *testing.T, expected interface{}, actual interface{}) {
	want, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("failed to marshal expected value to JSON: %v", err)
	}
	have, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("failed to marshal actual value to JSON: %v", err)
	}
	if !bytes.Equal(want, have) {
		t.Fatalf("incorrect supply info:\nhave: %s\nwant: %s", string(have), string(want))
	}
}
