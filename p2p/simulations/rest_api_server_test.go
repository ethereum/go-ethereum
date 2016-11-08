package simulations

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
)

const testPort = "8889"

type testController struct {
}

func (self *testController) SetResource(id string, c Controller) {
}

func (self *testController) Resource(id string) (Controller, error) {
	if id == "missing" {
		return nil, fmt.Errorf("missing")
	}
	return Controller(self), nil
}

func (self *testController) Handle(method string) (returnHandler, error) {
	switch method {
	case "POST":
	case "DELETE":
	default:
		return nil, fmt.Errorf("allowed methods: POST DELETE")
	}
	return handlerf(method), nil
}

func handlerf(method string) returnHandler {
	return func(r io.Reader) (io.ReadSeeker, error) {
		body, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		if string(body) == "invalid" {
			return nil, fmt.Errorf("invalid body")
		}
		return io.ReadSeeker(bytes.NewReader([]byte("response"))), nil
	}
}

func init() {
	StartRestApiServer(testPort, &testController{})
}

type testRequest struct {
	method   string
	path     string
	body     string
	response string
	status   int
}

type ReadCloser struct {
	io.Reader
}

func (ReadCloser) Close() {}

func testResponses(t *testing.T, reqs ...*testRequest) {
	for _, req := range reqs {
		path := url(testPort, req.path)
		var r *http.Response
		var err error
		switch req.method {
		case "POST":
			r, err = http.Post(path, "text/json", ReadCloser{bytes.NewReader([]byte(req.body))})
		default:
			r, err = http.Get(path)
		}
		if err != nil {
			t.Fatalf("unexpected error on request: %v", err)
		}
		if r.StatusCode != req.status {
			t.Fatalf("unexpected status on request: got %v, expected %v", r.StatusCode, req.status)
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("unexpected error on reading body: %v", err)
		}
		if string(body) != req.response {
			t.Fatalf("unexpected response body. got '%s', expected '%v'", body, req.response)
		}
	}
}

func TestServerMethodNotAllowed(t *testing.T) {
	testResponses(t,
		&testRequest{
			"GET",
			"anypath",
			"anybody",
			"method GET not allowed (allowed methods: POST DELETE)\n",
			http.StatusMethodNotAllowed,
		})
}

func TestServerInvalid(t *testing.T) {
	testResponses(t,
		&testRequest{
			"POST",
			"anypath",
			"invalid",
			"handler error: invalid body\n",
			http.StatusBadRequest,
		})
}

func TestServerResourceNotFound(t *testing.T) {
	testResponses(t,
		&testRequest{
			"POST",
			"missing",
			"anybody",
			"resource missing not found\n",
			http.StatusNotFound,
		})
}

func TestServerSuccess(t *testing.T) {
	testResponses(t,
		&testRequest{
			"POST",
			"anypath",
			"anybody",
			"response",
			http.StatusOK,
		})
}
