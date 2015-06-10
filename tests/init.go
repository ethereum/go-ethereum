package tests

import (
	"encoding/json"
	"io"
	"io/ioutil"
	// "log"
	"net/http"
	"os"
	"testing"

	// logpkg "github.com/ethereum/go-ethereum/logger"
)

// var Logger *logpkg.StdLogSystem
// var Log = logpkg.NewLogger("TEST")

// func init() {
// 	Logger = logpkg.NewStdLogSystem(os.Stdout, log.LstdFlags, logpkg.InfoLevel)
// 	logpkg.AddLogSystem(Logger)
// }

func readJSON(t *testing.T, reader io.Reader, value interface{}) {
	data, err := ioutil.ReadAll(reader)
	err = json.Unmarshal(data, &value)
	if err != nil {
		t.Error(err)
	}
}

func CreateHttpTests(t *testing.T, uri string, value interface{}) {
	resp, err := http.Get(uri)
	if err != nil {
		t.Error(err)

		return
	}
	defer resp.Body.Close()

	readJSON(t, resp.Body, value)
}

func CreateFileTests(t *testing.T, fn string, value interface{}) {
	file, err := os.Open(fn)
	if err != nil {
		t.Error(err)

		return
	}
	defer file.Close()

	readJSON(t, file, value)
}
