// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewID(t *testing.T) {
	hexchars := "0123456789ABCDEFabcdef"
	for i := 0; i < 100; i++ {
		id := string(NewID())
		if !strings.HasPrefix(id, "0x") {
			t.Fatalf("invalid ID prefix, want '0x...', got %s", id)
		}

		id = id[2:]
		if len(id) == 0 || len(id) > 32 {
			t.Fatalf("invalid ID length, want len(id) > 0 && len(id) <= 32), got %d", len(id))
		}

		for i := 0; i < len(id); i++ {
			if strings.IndexByte(hexchars, id[i]) == -1 {
				t.Fatalf("unexpected byte, want any valid hex char, got %c", id[i])
			}
		}
	}
}

func TestSubscriptions(t *testing.T) {
	var (
		namespaces        = []string{"eth", "shh", "bzz"}
		service           = &notificationTestService{}
		subCount          = len(namespaces)
		notificationCount = 3

		server                 = NewServer()
		clientConn, serverConn = net.Pipe()
		out                    = json.NewEncoder(clientConn)
		in                     = json.NewDecoder(clientConn)
		successes              = make(chan subConfirmation)
		notifications          = make(chan subscriptionResult)
		errors                 = make(chan error, subCount*notificationCount+1)
	)

	// setup and start server
	for _, namespace := range namespaces {
		if err := server.RegisterName(namespace, service); err != nil {
			t.Fatalf("unable to register test service %v", err)
		}
	}
	go server.ServeCodec(NewCodec(serverConn), 0)
	defer server.Stop()

	// wait for message and write them to the given channels
	go waitForMessages(in, successes, notifications, errors)

	// create subscriptions one by one
	for i, namespace := range namespaces {
		request := map[string]interface{}{
			"id":      i,
			"method":  fmt.Sprintf("%s_subscribe", namespace),
			"version": "2.0",
			"params":  []interface{}{"someSubscription", notificationCount, i},
		}
		if err := out.Encode(&request); err != nil {
			t.Fatalf("Could not create subscription: %v", err)
		}
	}

	timeout := time.After(30 * time.Second)
	subids := make(map[string]string, subCount)
	count := make(map[string]int, subCount)
	allReceived := func() bool {
		done := len(count) == subCount
		for _, c := range count {
			if c < notificationCount {
				done = false
			}
		}
		return done
	}
	for !allReceived() {
		select {
		case confirmation := <-successes: // subscription created
			subids[namespaces[confirmation.reqid]] = string(confirmation.subid)
		case notification := <-notifications:
			count[notification.ID]++
		case err := <-errors:
			t.Fatal(err)
		case <-timeout:
			for _, namespace := range namespaces {
				subid, found := subids[namespace]
				if !found {
					t.Errorf("subscription for %q not created", namespace)
					continue
				}
				if count, found := count[subid]; !found || count < notificationCount {
					t.Errorf("didn't receive all notifications (%d<%d) in time for namespace %q", count, notificationCount, namespace)
				}
			}
			t.Fatal("timed out")
		}
	}
}

// This test checks that unsubscribing works.
func TestServerUnsubscribe(t *testing.T) {
	// Start the server.
	server := newTestServer()
	service := &notificationTestService{unsubscribed: make(chan string)}
	server.RegisterName("nftest2", service)
	p1, p2 := net.Pipe()
	go server.ServeCodec(NewCodec(p1), 0)

	p2.SetDeadline(time.Now().Add(10 * time.Second))

	// Subscribe.
	p2.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"nftest2_subscribe","params":["someSubscription",0,10]}`))

	// Handle received messages.
	resps := make(chan subConfirmation)
	notifications := make(chan subscriptionResult)
	errors := make(chan error)
	go waitForMessages(json.NewDecoder(p2), resps, notifications, errors)

	// Receive the subscription ID.
	var sub subConfirmation
	select {
	case sub = <-resps:
	case err := <-errors:
		t.Fatal(err)
	}

	// Unsubscribe and check that it is handled on the server side.
	p2.Write([]byte(`{"jsonrpc":"2.0","method":"nftest2_unsubscribe","params":["` + sub.subid + `"]}`))
	for {
		select {
		case id := <-service.unsubscribed:
			if id != string(sub.subid) {
				t.Errorf("wrong subscription ID unsubscribed")
			}
			return
		case err := <-errors:
			t.Fatal(err)
		case <-notifications:
			// drop notifications
		}
	}
}

type subConfirmation struct {
	reqid int
	subid ID
}

func waitForMessages(in *json.Decoder, successes chan subConfirmation, notifications chan subscriptionResult, errors chan error) {
	for {
		var msg jsonrpcMessage
		if err := in.Decode(&msg); err != nil {
			errors <- fmt.Errorf("decode error: %v", err)
			return
		}
		switch {
		case msg.isNotification():
			var res subscriptionResult
			if err := json.Unmarshal(msg.Params, &res); err != nil {
				errors <- fmt.Errorf("invalid subscription result: %v", err)
			} else {
				notifications <- res
			}
		case msg.isResponse():
			var c subConfirmation
			if msg.Error != nil {
				errors <- msg.Error
			} else if err := json.Unmarshal(msg.Result, &c.subid); err != nil {
				errors <- fmt.Errorf("invalid response: %v", err)
			} else {
				json.Unmarshal(msg.ID, &c.reqid)
				successes <- c
			}
		default:
			errors <- fmt.Errorf("unrecognized message: %v", msg)
			return
		}
	}
}
