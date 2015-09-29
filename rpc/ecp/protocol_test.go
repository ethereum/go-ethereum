package ecp

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"strings"
	"testing"
)

type testReader struct {
	r io.Reader
}

func (r *testReader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *testReader) Close() error {
	return nil
}

func newTestReader(source string) io.ReadCloser {
	return &testReader{strings.NewReader(source)}
}

func TestNoArgsAndRets(t *testing.T) {
	r := newTestReader("*1\r\n+Service.Method\r\n")
	codec := NewECPCodec(r, nil)

	req, err := codec.Read()
	if err != nil {
		t.Fatalf("%s\n", err)
	}

	if req.service != "Service" {
		t.Fatalf("Expected service 'Service' but got '%s'\n", req.service)
	}

	if req.method != "Method" {
		t.Fatalf("Expected method 'Method' but got '%s'\n", req.method)
	}
}

func TestWithArgs(t *testing.T) {
	i := big.NewInt(123456789)
	ib := i.Bytes()
	str := "some string"
	bstr := []byte{'a', '\r', '\n', 'b'}
	in := int64(42)

	r := newTestReader(fmt.Sprintf("*5\r\n+Service.Method\r\n$%d\r\n%s\r\n+%s\r\n$%d\r\n%s\r\n:%d\r\n",
		len(ib), ib, str, len(bstr), bstr, in))

	codec := NewECPCodec(r, nil)

	req, err := codec.Read()
	if err != nil {
		t.Fatalf("%s\n", err)
	}

	if req.service != "Service" {
		t.Fatalf("Expected service 'Service' but got '%s'\n", req.service)
	}

	if req.method != "Method" {
		t.Fatalf("Expected method 'Method' but got '%s'\n", req.method)
	}

	if len(req.args) != 4 {
		t.Fatalf("Expected 3 args, got %d args(s)\n", len(req.args))
	}

	if pin, ok := req.args[0].([]byte); ok {
		parsedInt := new(big.Int).SetBytes(pin)
		if parsedInt.Cmp(i) != 0 {
			t.Fatalf("Expected big.Int(%d).Cmp(%d) == 0\n", parsedInt, i)
		}
	} else {
		t.Fatalf("Expected arg[0] to be []byte, got %T\n", req.args[0])
	}

	if s, ok := req.args[1].(string); ok {
		if str != s {
			t.Fatalf("Expected 'some string' == '%s'\n", s)
		}
	} else {
		t.Fatalf("Expected arg[1] to be string, got %T\n", req.args[1])
	}

	if arr, ok := req.args[2].([]byte); ok {
		if bytes.Compare(arr, bstr) != 0 {
			t.Fatalf("Expected %s, got %s\n", bstr, arr)
		}
	} else {
		t.Fatalf("Expected arg[2] to be []byte, got %T\n", req.args[2])
	}

	if val, ok := req.args[3].(int64); ok {
		if in != val {
			t.Fatalf("Expected %d, got %d\n", in, val)
		}
	} else {
		t.Fatalf("Expected arg[3] to be int64, got %T\n", req.args[3])
	}
}
