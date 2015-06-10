package tests

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

var (
	baseDir            = filepath.Join(".", "files")
	blockTestDir       = filepath.Join(baseDir, "BlockTests")
	stateTestDir       = filepath.Join(baseDir, "StateTests")
	transactionTestDir = filepath.Join(baseDir, "TransactionTests")
	vmTestDir          = filepath.Join(baseDir, "VMTests")

	blockSkipTests = []string{}
	transSkipTests = []string{"TransactionWithHihghNonce256"}
	stateSkipTests = []string{"mload32bitBound_return", "mload32bitBound_return2"}
	vmSkipTests    = []string{}
)

func readJSON(reader io.Reader, value interface{}) error {
	data, err := ioutil.ReadAll(reader)
	err = json.Unmarshal(data, &value)
	if err != nil {
		return err
	}
	return nil
}

func CreateHttpTests(uri string, value interface{}) error {
	resp, err := http.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = readJSON(resp.Body, value)
	if err != nil {
		return err
	}
	return nil
}

func CreateFileTests(fn string, value interface{}) error {
	file, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer file.Close()

	err = readJSON(file, value)
	if err != nil {
		return err
	}
	return nil
}
