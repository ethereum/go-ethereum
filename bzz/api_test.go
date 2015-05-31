package bzz

import (
	// "net/http"
	// "strings"
	"testing"
	// "time"
	"os"
	"path"
	"runtime"

	// "github.com/ethereum/go-ethereum/common/docserver"
)

var (
	testDir string
	datadir = "/tmp/bzz"
)

func init() {
	_, filename, _, _ := runtime.Caller(1)
	testDir = path.Join(path.Dir(filename), "bzztest")
}

func testApi() (api *Api, err error) {
	os.RemoveAll(datadir)
	api, err = NewLocalApi(datadir)
	if err != nil {
		return
	}
	api.Start(nil, nil)
	return
}

func TestApiPut(t *testing.T) {
	api, err := testApi()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expContent := "hello"
	expMimeType := "text/plain"
	expStatus := 0
	expSize := len(expContent)
	bzzhash, err := api.Put(expContent, expMimeType)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	bytecontent, mimeType, status, size, err := api.Get(bzzhash)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	content := string(bytecontent)
	if content != expContent {
		t.Errorf("incorrect content. expected '%s', got '%s'", expContent, content)
	}
	if mimeType != expMimeType {
		t.Errorf("incorrect mimeType. expected '%s', got '%s'", expMimeType, mimeType)
	}
	if status != expStatus {
		t.Errorf("incorrect status. expected '%s', got '%s'", expStatus, status)
	}
	if size != expSize {
		t.Errorf("incorrect size. expected '%d', got '%d'", expSize, size)
	}
}

func TestApiUpload(t *testing.T) {
	api, err := testApi()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_ = api
}
