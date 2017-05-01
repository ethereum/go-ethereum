package simulations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

/***
 * \todo rewrite this with a scripting engine to do http protocol xchanges more easily
 */
const (
	domain = "http://localhost"
	port   = "8888"
)

var quitc chan bool
var controller *ResourceController

func init() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlError, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	controller, quitc = NewSessionController(DefaultNet)
	StartRestApiServer(port, controller)
}

func url(port, path string) string {
	return fmt.Sprintf("%v:%v/%v", domain, port, path)
}

func TestDelete(t *testing.T) {
	req, err := http.NewRequest("DELETE", url(port, ""), nil)
	if err != nil {
		t.Fatalf("unexpected error")
	}
	var resp *http.Response
	go func() {
		r, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("unexpected error")
		}
		resp = r
	}()
	timeout := time.NewTimer(1000 * time.Millisecond)
	select {
	case <-quitc:
	case <-timeout.C:
		t.Fatalf("timed out: controller did not quit, response: %v", resp)
	}
}

func TestCreate(t *testing.T) {
	s, err := json.Marshal(&struct{ Id string }{Id: "testnetwork"})
	req, err := http.NewRequest("POST", domain+":"+port, bytes.NewReader(s))
	if err != nil {
		t.Fatalf("unexpected error creating request: %v", err)
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("unexpected error on http.Client request: %v", err)
	}
	req, err = http.NewRequest("POST", domain+":"+port+"/testnetwork/debug/", nil)
	if err != nil {
		t.Fatalf("unexpected error creating request: %v", err)
	}
	resp, err = (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("unexpected error on http.Client request: %v", err)
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading response body: %v", err)
	}
}

func TestNodes(t *testing.T) {
	networkname := "testnetworkfornodes"

	s, err := json.Marshal(&struct{ Id string }{Id: networkname})
	req, err := http.NewRequest("POST", domain+":"+port, bytes.NewReader(s))
	if err != nil {
		t.Fatalf("unexpected error creating request: %v", err)
	}
	_, err = (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("unexpected error on http.Client request: %v", err)
	}
	for i := 0; i < 3; i++ {
		req, err = http.NewRequest("POST", domain+":"+port+"/"+networkname+"/node/", nil)
		if err != nil {
			t.Fatalf("unexpected error creating request: %v", err)
		}
		_, err = (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("unexpected error on http.Client request: %v", err)
		}
	}
}

func testResponse(t *testing.T, method, addr string, r io.ReadSeeker) []byte {

	req, err := http.NewRequest(method, addr, r)
	if err != nil {
		t.Fatalf("unexpected error creating request: %v", err)
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("unexpected error on http.Client request: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading response body: %v", err)
	}
	return body

}

func TestUpdate(t *testing.T) {
	t.Skip("...")
	conf := &NetworkConfig{
		Id:      "0",
		Backend: false,
	}
	mc := NewNetworkController(NewNetwork(conf), nil)
	controller.SetResource(conf.Id, mc)
	exp := `{
  "add": [
    {
      "data": {
        "id": "aa7c",
        "up": true
      },
      "group": "nodes"
    },
    {
      "data": {
        "id": "f5ae",
        "up": true
      },
      "group": "nodes"
    }
  ],
  "remove": [],
  "message": []
}`
	s, _ := json.Marshal(&SimConfig{})
	resp := testResponse(t, "GET", url(port, "0"), bytes.NewReader(s))
	if string(resp) != exp {
		t.Fatalf("incorrect response body. got\n'%v', expected\n'%v'", string(resp), exp)
	}
}

func mockNewNodes(eventer *event.TypeMux, ids []*adapters.NodeId) {
	log.Trace("mock starting")
	for _, id := range ids {
		log.Trace(fmt.Sprintf("mock adding node %v", id))
		eventer.Post(&NodeEvent{
			Action: "up",
			Type:   "node",
			node:   &Node{Id: id, config: &NodeConfig{Id: id}},
		})
	}
}
