package helper

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func CreateTests(uri string, value interface{}) error {
	resp, err := http.Get(uri)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	return nil
}
