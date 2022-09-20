package api

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type testFetcher struct {
	response string
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func makeApi(response string) *BeaconLightApi {
	return &BeaconLightApi{client: &testFetcher{response}}
}

func (f *testFetcher) Do(req *http.Request) (*http.Response, error) {
	res := new(http.Response)
	res.StatusCode = 200
	res.Body = nopCloser{bytes.NewBufferString(f.response)}
	return res, nil
}

func TestGetCheckpointData(t *testing.T) {
	resp :=
		`{ "data": 
	{ 
	  "header": {}
	}
}`
	_, err := makeApi(resp).GetCheckpointData(common.HexToHash("0xc78009fdf07fc56a11f122370658a353aaa542ed63e44c4bc15ff4cd105ab33c"))
	t.Logf("err: %v", err)
	//TODO finish this test
}
