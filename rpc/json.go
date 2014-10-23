package rpc

import (
	"encoding/json"
	"io"
)

type jsonWrapper struct{}

func (self jsonWrapper) Send(writer io.Writer, v interface{}) (n int, err error) {
	var payload []byte
	payload, err = json.Marshal(v)
	if err != nil {
		return 0, err
	}

	return writer.Write(payload)
}

var JSON jsonWrapper
