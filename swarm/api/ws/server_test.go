package ws

import (
	"context"
	"testing"
	"time"
	
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc"
)

func init() {
	glog.SetV(logger.Detail)
	glog.SetToStderr(true)
}

type TestResult struct {
	Foo string `json:"foo"`
}

func TestStartWSServer(t *testing.T) {
	ep := "localhost:8099"
	server := &Server{
		Endpoint: ep,
		CorsString: "*",
	}
	apis := []rpc.API{
		{
			Namespace: "pss",
			Version:   "0.1",
			Service:   makeFakeAPIHandler(),
			Public:    true,
		},
	}
	go func() {
		err := StartWSServer(apis, server)
		t.Logf("wsserver exited: %v", err)
	}()
	
	time.Sleep(time.Second)
	
	client, err := rpc.DialWebsocket(context.Background(), "ws://" + ep, "ws://localhost")
	if err != nil {
		t.Fatalf("could not connect: %v", err)
	} else {
		t.Logf("client: %v", client)
		client.Call(&TestResult{}, "pss_test")
	}
	
}

func makeFakeAPIHandler() *FakeAPIHandler {
	return &FakeAPIHandler{}
}

type FakeAPIHandler struct {
}

func (self *FakeAPIHandler) Test() {
	glog.V(logger.Detail).Infof("in fakehandler Test()")
}
