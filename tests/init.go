package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	// "log"
	"net/http"
	"os"

	// logpkg "github.com/ethereum/go-ethereum/logger"
)

// var Logger *logpkg.StdLogSystem
// var Log = logpkg.NewLogger("TEST")

// func init() {
// 	Logger = logpkg.NewStdLogSystem(os.Stdout, log.LstdFlags, logpkg.InfoLevel)
// 	logpkg.AddLogSystem(Logger)
// }

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
}
