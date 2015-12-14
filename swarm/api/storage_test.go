package api

import (
	"testing"
)

func testStorage(t *testing.T, f func(*Storage)) {
	testApi(t, func(api *Api) {
		f(NewStorage(api))
	})
}

func TestStoragePutGet(t *testing.T) {
	testStorage(t, func(api *Storage) {
		content := "hello"
		exp := expResponse(content, "text/plain", 0)
		// exp := expResponse([]byte(content), "text/plain", 0)
		bzzhash, err := api.Put(content, exp.MimeType)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// to check put against the Api#Get
		resp0 := testGet(t, api.api, bzzhash)
		checkResponse(t, resp0, exp)

		// check storage#Get
		resp, err := api.Get(bzzhash)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		checkResponse(t, &testResponse{nil, resp}, exp)
	})
}
