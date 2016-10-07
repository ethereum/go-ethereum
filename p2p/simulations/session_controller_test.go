package simulations

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

const (
	domain = "http://localhost"
	port   = "8888"
)

var quitc chan bool
var controller *ResourceController

func init() {
	glog.SetV(6)
	glog.SetToStderr(true)
	controller, quitc = NewSessionController()
	StartRestApiServer(port, controller)
}

func url(port, path string) string {
	return fmt.Sprintf("%v:%v/%v", domain, port, path)
}

func TestQuit(t *testing.T) {
	req, err := http.NewRequest("DELETE", url(port, ""), nil)
	// req, err := http.NewRequest("PUT", url(""), nil)
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

func TestUpdate(t *testing.T) {
	req, err := http.NewRequest("GET", url(port, "0"), nil)
	if err != nil {
		t.Fatalf("unexpected error")
	}
	keys := []string{
		"aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80",
		"f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3",
	}
	var ids []*discover.NodeID
	for _, key := range keys {
		id := discover.MustHexID(key)
		ids = append(ids, &id)
	}
	network := NewNetwork(nil)
	NewNetworkController(network, controller)
	Update(network, ids)
	r, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("unexpected error")
	}
	resp, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("error reading response body: %v", err)
	}
	exp := `{
  "add": [
    {
      "data": {
        "id": "aa7c",
        "source": "",
        "target": "",
        "on": false
      },
      "classes": "",
      "group": "nodes"
    },
    {
      "data": {
        "id": "f5ae",
        "source": "",
        "target": "",
        "on": false
      },
      "classes": "",
      "group": "nodes"
    }
  ],
  "remove": []
}`
	if string(resp) != exp {
		t.Fatalf("incorrect response body. got\n'%v', expected\n'%v'", string(resp), exp)
	}
}

func Update(self *Network, ids []*discover.NodeID) {
	self.lock.Lock()
	defer self.lock.Unlock()
	for _, id := range ids {
		e := &Entry{
			Action: "Add",
			Type:   "Node",
			Object: &SimNode{ID: id, config: &NodeConfig{ID: id}},
		}
		self.Journal = append(self.Journal, e)
	}
}
