package helper

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
)

func CreateTests(t *testing.T, uri string, value interface{}) {
	resp, err := http.Get(uri)
	if err != nil {
		t.Error(err)

		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(data, &value)
	if err != nil {
		t.Error(err)
	}
}
