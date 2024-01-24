package request

import (
	"reflect"
	"testing"
)

func TestEventFilter(t *testing.T) {
	s := NewScheduler()
	module1 := &testModule{name: "module1"}
	module2 := &testModule{name: "module2"}
	s.RegisterModule(module1, "module1")
	s.RegisterModule(module2, "module2")
	s.Start()
	// startup process round without events
	s.testWaitCh <- struct{}{}
	module1.expProcess(t, nil)
	module2.expProcess(t, nil)
	srv := &testServer{}
	// register server; both modules should receive server event
	s.RegisterServer(srv)
	s.testWaitCh <- struct{}{}
	module1.expProcess(t, []Event{
		Event{Type: EvRegistered, Server: srv},
	})
	module2.expProcess(t, []Event{
		Event{Type: EvRegistered, Server: srv},
	})
	// let module1 send a request
	srv.canRequest = 1
	module1.reqc = testRequest
	s.Trigger()
	// first triggered round sends the request, no events yet
	s.testWaitCh <- struct{}{}
	module1.expProcess(t, nil)
	module2.expProcess(t, nil)
	// next round triggered by EvRequest; only module1 should receive it
	s.testWaitCh <- struct{}{}
	module1.expProcess(t, []Event{
		Event{Type: EvRequest, Server: srv, Data: RequestResponse{ID: 1, Request: testRequest}},
	})
	module2.expProcess(t, nil)
	// server emits EvTimeout; only module1 should receive it
	srv.eventCb(Event{Type: EvTimeout, Data: RequestResponse{ID: 1, Request: testRequest}})
	s.testWaitCh <- struct{}{}
	module1.expProcess(t, []Event{
		Event{Type: EvTimeout, Server: srv, Data: RequestResponse{ID: 1, Request: testRequest}},
	})
	module2.expProcess(t, nil)
	// unregister server; both modules should receive server event
	s.UnregisterServer(srv)
	s.testWaitCh <- struct{}{}
	module1.expProcess(t, []Event{
		// module1 should also receive EvFail on its pending request
		Event{Type: EvFail, Server: srv, Data: RequestResponse{ID: 1, Request: testRequest}},
		Event{Type: EvUnregistered, Server: srv},
	})
	module2.expProcess(t, []Event{
		Event{Type: EvUnregistered, Server: srv},
	})
	// response after server unregistered; should be discarded
	srv.eventCb(Event{Type: EvResponse, Data: RequestResponse{ID: 1, Request: testRequest, Response: testResponse}})
	s.testWaitCh <- struct{}{}
	module1.expProcess(t, nil)
	module2.expProcess(t, nil)
	// no more process rounds expected; shut down
	s.testWaitCh <- struct{}{}
	module1.expNoMoreProcess(t)
	module2.expNoMoreProcess(t)
	s.Stop()
}

type testServer struct {
	eventCb    func(Event)
	lastID     ID
	canRequest int
}

func (s *testServer) subscribe(eventCb func(Event)) {
	s.eventCb = eventCb
}

func (s *testServer) canRequestNow() (bool, float32) {
	return s.canRequest > 0, 0
}

func (s *testServer) sendRequest(req Request) ID {
	s.canRequest--
	s.lastID++
	s.eventCb(Event{Type: EvRequest, Data: RequestResponse{ID: s.lastID, Request: req}})
	return s.lastID
}

func (s *testServer) Fail(string)  {}
func (s *testServer) unsubscribe() {}

type testModule struct {
	name      string
	processed [][]Event
	reqc      Request // request candidate
}

func (m *testModule) Process(events []Event) {
	m.processed = append(m.processed, events)
}

func (m *testModule) MakeRequest(Server) (Request, float32) {
	return m.reqc, 0
}

func (m *testModule) expProcess(t *testing.T, expEvents []Event) {
	if len(m.processed) == 0 {
		t.Errorf("Missing call to %s.Process", m.name)
		return
	}
	events := m.processed[0]
	m.processed = m.processed[1:]
	if !reflect.DeepEqual(events, expEvents) {
		t.Errorf("Call to %s.Process with wrong events (expected %v, got %v)", m.name, expEvents, events)
	}
}

func (m *testModule) expNoMoreProcess(t *testing.T) {
	for len(m.processed) > 0 {
		t.Errorf("Unexpected call to %s.Process with events %v", m.name, m.processed[0])
		m.processed = m.processed[1:]
	}
}
