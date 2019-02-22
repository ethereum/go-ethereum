// Copyright 2015 The go-ethereum Authors
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

package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/ubiq/go-ubiq/params"
)

var (
	baseDir            = filepath.Join(".", "testdata")
	blockTestDir       = filepath.Join(baseDir, "BlockchainTests")
	stateTestDir       = filepath.Join(baseDir, "GeneralStateTests")
	transactionTestDir = filepath.Join(baseDir, "TransactionTests")
	vmTestDir          = filepath.Join(baseDir, "VMTests")
	rlpTestDir         = filepath.Join(baseDir, "RLPTests")
)

func readJson(reader io.Reader, value interface{}) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("error reading JSON file: %v", err)
	}
	if err = json.Unmarshal(data, &value); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(data, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at line %v: %v", line, err)
		}
		return err
	}
	return nil
}

func readJsonFile(fn string, value interface{}) error {
	file, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer file.Close()

	err = readJson(file, value)
	if err != nil {
		return fmt.Errorf("%s in file %s", err.Error(), fn)
	}
	return nil
}

// findLine returns the line number for the given offset into data.
func findLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
}

// testMatcher controls skipping and chain config assignment to tests.
type testMatcher struct {
	configpat    []testConfig
	failpat      []testFailure
	skiploadpat  []*regexp.Regexp
	skipshortpat []*regexp.Regexp
}

type testConfig struct {
	p      *regexp.Regexp
	config params.ChainConfig
}

type testFailure struct {
	p      *regexp.Regexp
	reason string
}

// skipShortMode skips tests matching when the -short flag is used.
func (tm *testMatcher) skipShortMode(pattern string) {
	tm.skipshortpat = append(tm.skipshortpat, regexp.MustCompile(pattern))
}

// skipLoad skips JSON loading of tests matching the pattern.
func (tm *testMatcher) skipLoad(pattern string) {
	tm.skiploadpat = append(tm.skiploadpat, regexp.MustCompile(pattern))
}

// fails adds an expected failure for tests matching the pattern.
func (tm *testMatcher) fails(pattern string, reason string) {
	if reason == "" {
		panic("empty fail reason")
	}
	tm.failpat = append(tm.failpat, testFailure{regexp.MustCompile(pattern), reason})
}

// config defines chain config for tests matching the pattern.
func (tm *testMatcher) config(pattern string, cfg params.ChainConfig) {
	tm.configpat = append(tm.configpat, testConfig{regexp.MustCompile(pattern), cfg})
}

// findSkip matches name against test skip patterns.
func (tm *testMatcher) findSkip(name string) (reason string, skipload bool) {
	if testing.Short() {
		for _, re := range tm.skipshortpat {
			if re.MatchString(name) {
				return "skipped in -short mode", false
			}
		}
	}
	for _, re := range tm.skiploadpat {
		if re.MatchString(name) {
			return "skipped by skipLoad", true
		}
	}
	return "", false
}

// findConfig returns the chain config matching defined patterns.
func (tm *testMatcher) findConfig(name string) *params.ChainConfig {
	// TODO(fjl): name can be derived from testing.T when min Go version is 1.8
	for _, m := range tm.configpat {
		if m.p.MatchString(name) {
			return &m.config
		}
	}
	return new(params.ChainConfig)
}

// checkFailure checks whether a failure is expected.
func (tm *testMatcher) checkFailure(t *testing.T, name string, err error) error {
	// TODO(fjl): name can be derived from t when min Go version is 1.8
	failReason := ""
	for _, m := range tm.failpat {
		if m.p.MatchString(name) {
			failReason = m.reason
			break
		}
	}
	if failReason != "" {
		t.Logf("expected failure: %s", failReason)
		if err != nil {
			t.Logf("error: %v", err)
			return nil
		} else {
			return fmt.Errorf("test succeeded unexpectedly")
		}
	}
	return err
}

// walk invokes its runTest argument for all subtests in the given directory.
//
// runTest should be a function of type func(t *testing.T, name string, x <TestType>),
// where TestType is the type of the test contained in test files.
func (tm *testMatcher) walk(t *testing.T, dir string, runTest interface{}) {
	// Walk the directory.
	dirinfo, err := os.Stat(dir)
	if os.IsNotExist(err) || !dirinfo.IsDir() {
		fmt.Fprintf(os.Stderr, "can't find test files in %s, did you clone the tests submodule?\n", dir)
		t.Skip("missing test files")
	}
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		name := filepath.ToSlash(strings.TrimPrefix(path, dir+string(filepath.Separator)))
		if info.IsDir() {
			if _, skipload := tm.findSkip(name + "/"); skipload {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".json" {
			t.Run(name, func(t *testing.T) { tm.runTestFile(t, path, name, runTest) })
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (tm *testMatcher) runTestFile(t *testing.T, path, name string, runTest interface{}) {
	if r, _ := tm.findSkip(name); r != "" {
		t.Skip(r)
	}
	t.Parallel()

	// Load the file as map[string]<testType>.
	m := makeMapFromTestFunc(runTest)
	if err := readJsonFile(path, m.Addr().Interface()); err != nil {
		t.Fatal(err)
	}

	// Run all tests from the map. Don't wrap in a subtest if there is only one test in the file.
	keys := sortedMapKeys(m)
	if len(keys) == 1 {
		runTestFunc(runTest, t, name, m, keys[0])
	} else {
		for _, key := range keys {
			name := name + "/" + key
			t.Run(key, func(t *testing.T) {
				if r, _ := tm.findSkip(name); r != "" {
					t.Skip(r)
				}
				runTestFunc(runTest, t, name, m, key)
			})
		}
	}
}

func makeMapFromTestFunc(f interface{}) reflect.Value {
	stringT := reflect.TypeOf("")
	testingT := reflect.TypeOf((*testing.T)(nil))
	ftyp := reflect.TypeOf(f)
	if ftyp.Kind() != reflect.Func || ftyp.NumIn() != 3 || ftyp.NumOut() != 0 || ftyp.In(0) != testingT || ftyp.In(1) != stringT {
		panic(fmt.Sprintf("bad test function type: want func(*testing.T, string, <TestType>), have %s", ftyp))
	}
	testType := ftyp.In(2)
	mp := reflect.New(reflect.MapOf(stringT, testType))
	return mp.Elem()
}

func sortedMapKeys(m reflect.Value) []string {
	keys := make([]string, m.Len())
	for i, k := range m.MapKeys() {
		keys[i] = k.String()
	}
	sort.Strings(keys)
	return keys
}

func runTestFunc(runTest interface{}, t *testing.T, name string, m reflect.Value, key string) {
	reflect.ValueOf(runTest).Call([]reflect.Value{
		reflect.ValueOf(t),
		reflect.ValueOf(name),
		m.MapIndex(reflect.ValueOf(key)),
	})
}
