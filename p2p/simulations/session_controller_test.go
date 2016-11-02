package simulations

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
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

	keys := []string{
		"aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80",
		"f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3",
	}
	var ids []*discover.NodeID
	for _, key := range keys {
		id := discover.MustHexID(key)
		ids = append(ids, &id)
	}

	eventer := &event.TypeMux{}
	journal := NewJournal()
	journal.Subscribe(eventer, &Entry{})
	mockNewNodes(eventer, ids)
	conf := &NetworkConfig{
		Id: "0",
	}
	mc := NewNetworkController(conf, eventer, journal)
	controller.SetResource(conf.Id, mc)
	journal.WaitEntries(len(ids))

	req, err := http.NewRequest("GET", url(port, "0"), bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("unexpected error creating request: %v", err)
	}
	r, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("unexpected error on http.Client request: %v", err)
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

func mockNewNodes(eventer *event.TypeMux, ids []*discover.NodeID) {
	glog.V(6).Infof("mock starting")
	for _, id := range ids {
		glog.V(6).Infof("mock adding node %v", id)
		eventer.Post(&Entry{
			Action: "Add",
			Type:   "Node",
			Object: &SimNode{ID: id, config: &NodeConfig{ID: id}},
		})
	}
}

// func TestReplay(t *testing.T) {

// }
