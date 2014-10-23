package helper

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

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
