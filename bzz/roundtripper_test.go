package bzz

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/docserver"
)

const port = "8500"

func TestRoundTripper(t *testing.T) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/plain")
			http.ServeContent(w, r, "", time.Unix(0, 0), strings.NewReader(r.RequestURI))
		} else {
			http.Error(w, "Method "+r.Method+" is not supported.", http.StatusMethodNotAllowed)
		}
	})
	go http.ListenAndServe(":"+port, nil)

	rt := &RoundTripper{port}
	ds := docserver.New("/")
	ds.RegisterProtocol("bzz", rt)

	resp, err := ds.Client().Get("bzz://test.com/path")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}

	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}
	if string(content) != "/test.com/path" {
		t.Errorf("incorrect response from http server: expected '%v', got '%v'", "/test.com/path", string(content))
	}

}
