package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/core"
)

var (
	baseDir            = filepath.Join(".", "files")
	blockTestDir       = filepath.Join(baseDir, "BlockchainTests")
	stateTestDir       = filepath.Join(baseDir, "StateTests")
	transactionTestDir = filepath.Join(baseDir, "TransactionTests")
	vmTestDir          = filepath.Join(baseDir, "VMTests")

	BlockSkipTests = []string{
		// Fails in InsertPreState with: computed state root does not
		// match genesis block bba25a96 0d8f85c8 Christoph said it will be
		// fixed eventually
		"SimpleTx3",

		// These tests are not valid, as they are out of scope for RLP and
		// the consensus protocol.
		"BLOCK__RandomByteAtTheEnd",
		"TRANSCT__RandomByteAtTheEnd",
		"BLOCK__ZeroByteAtTheEnd",
		"TRANSCT__ZeroByteAtTheEnd",
	}

	/* Go does not support transaction (account) nonces above 2^64. This
	technically breaks consensus but is regarded as "reasonable
	engineering constraint" as accounts cannot easily reach such high
	nonce values in practice
	*/
	TransSkipTests = []string{"TransactionWithHihghNonce256"}
	StateSkipTests = []string{}
	VmSkipTests    = []string{}
)

func readJson(reader io.Reader, value interface{}) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("Error reading JSON file", err.Error())
	}

	core.DisableBadBlockReporting = true
	if err = json.Unmarshal(data, &value); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(data, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at line %v: %v", line, err)
		}
		return fmt.Errorf("JSON unmarshal error: %v", err)
	}
	return nil
}

func readJsonHttp(uri string, value interface{}) error {
	resp, err := http.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = readJson(resp.Body, value)
	if err != nil {
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
