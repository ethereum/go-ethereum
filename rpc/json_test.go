package rpc

import (
	"bufio"
	"bytes"
	"reflect"
	"testing"
)

type RWC struct {
	*bufio.ReadWriter
}

func (rwc *RWC) Close() error {
	return nil
}

func TestJSONRequestParsing(t *testing.T) {
	server := NewServer()
	service := new(Service)

	if err := server.RegisterName("calc", service); err != nil {
		t.Fatalf("%v", err)
	}

	req := bytes.NewBufferString(`{"id": 1234, "jsonrpc": "2.0", "method": "calc_add", "params": [11, 22]}`)
	var str string
	reply := bytes.NewBufferString(str)
	rw := &RWC{bufio.NewReadWriter(bufio.NewReader(req), bufio.NewWriter(reply))}

	codec := NewJSONCodec(rw)

	requests, batch, err := codec.ReadRequestHeaders()
	if err != nil {
		t.Fatalf("%v", err)
	}

	if batch {
		t.Fatalf("Request isn't a batch")
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request but got %d requests - %v", len(requests), requests)
	}

	if requests[0].service != "calc" {
		t.Fatalf("Expected service 'calc' but got '%s'", requests[0].service)
	}

	if requests[0].method != "add" {
		t.Fatalf("Expected method 'Add' but got '%s'", requests[0].method)
	}

	if requests[0].id != 1234 {
		t.Fatalf("Expected id 1234 but got %d", requests[0].id)
	}

	var arg int
	args := []reflect.Type{reflect.TypeOf(arg), reflect.TypeOf(arg)}

	v, err := codec.ParseRequestArguments(args, requests[0].params)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if len(v) != 2 {
		t.Fatalf("Expected 2 argument values, got %d", len(v))
	}

	if v[0].Int() != 11 || v[1].Int() != 22 {
		t.Fatalf("expected %d == 11 && %d == 22", v[0].Int(), v[1].Int())
	}
}
