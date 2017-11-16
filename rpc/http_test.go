package rpc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPErrorResponseWithDelete(t *testing.T) {
	httpErrorResponseTest(t, "DELETE", contentType, "", http.StatusMethodNotAllowed)
}

func TestHTTPErrorResponseWithPut(t *testing.T) {
	httpErrorResponseTest(t, "PUT", contentType, "", http.StatusMethodNotAllowed)
}

func TestHTTPErrorResponseWithMaxContentLength(t *testing.T) {
	body := make([]rune, maxHTTPRequestContentLength+1, maxHTTPRequestContentLength+1)
	httpErrorResponseTest(t,
		"POST", contentType, string(body), http.StatusRequestEntityTooLarge)
}

func TestHTTPErrorResponseWithEmptyContentType(t *testing.T) {
	httpErrorResponseTest(t, "POST", "", "", http.StatusUnsupportedMediaType)
}

func TestHTTPErrorResponseWithValidRequest(t *testing.T) {
	httpErrorResponseTest(t, "POST", contentType, "", 0)
}

func httpErrorResponseTest(t *testing.T,
	method, contentType, body string, expectedResponse int) {

	request := httptest.NewRequest(method, "http://url.com", strings.NewReader(body))
	request.Header.Set("content-type", contentType)
	if response, _ := httpErrorResponse(request); response != expectedResponse {
		t.Fatalf("response code should be %d not %d", expectedResponse, response)
	}
}
