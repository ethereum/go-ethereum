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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
		namespaces        = []string{"eth", "bzz"}
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
			"jsonrpc": "2.0",
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
	p1, p2 := net.Pipe()
	defer p2.Close()

	// Start the server.
	server := newTestServer()
	service := &notificationTestService{unsubscribed: make(chan string, 1)}
	server.RegisterName("nftest2", service)
	go server.ServeCodec(NewCodec(p1), 0)

	// Subscribe.
	p2.SetDeadline(time.Now().Add(10 * time.Second))
	p2.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"nftest2_subscribe","params":["someSubscription",0,10]}`))

	// Handle received messages.
	var (
		resps         = make(chan subConfirmation)
		notifications = make(chan subscriptionResult)
		errors        = make(chan error, 1)
	)
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

// waitForMessages reads RPC messages from 'in' and dispatches them into the given channels.
// It stops if there is an error.
func waitForMessages(in *json.Decoder, successes chan subConfirmation, notifications chan subscriptionResult, errors chan error) {
	for {
		resp, notification, err := readAndValidateMessage(in)
		if err != nil {
			errors <- err
			return
		} else if resp != nil {
			successes <- *resp
		} else {
			notifications <- *notification
		}
	}
}

func readAndValidateMessage(in *json.Decoder) (*subConfirmation, *subscriptionResult, error) {
	var msg jsonrpcMessage
	if err := in.Decode(&msg); err != nil {
		return nil, nil, fmt.Errorf("decode error: %v", err)
	}
	switch {
	case msg.isNotification():
		var res subscriptionResult
		if err := json.Unmarshal(msg.Params, &res); err != nil {
			return nil, nil, fmt.Errorf("invalid subscription result: %v", err)
		}
		return nil, &res, nil
	case msg.isResponse():
		var c subConfirmation
		if msg.Error != nil {
			return nil, nil, msg.Error
		} else if err := json.Unmarshal(msg.Result, &c.subid); err != nil {
			return nil, nil, fmt.Errorf("invalid response: %v", err)
		} else {
			json.Unmarshal(msg.ID, &c.reqid)
			return &c, nil, nil
		}
	default:
		return nil, nil, fmt.Errorf("unrecognized message: %v", msg)
	}
}

type mockConn struct {
	enc *json.Encoder
}

// writeJSON writes a message to the connection.
func (c *mockConn) writeJSON(ctx context.Context, msg interface{}, isError bool) error {
	return c.enc.Encode(msg)
}

// closed returns a channel which is closed when the connection is closed.
func (c *mockConn) closed() <-chan interface{} { return nil }

// remoteAddr returns the peer address of the connection.
func (c *mockConn) remoteAddr() string { return "" }

// BenchmarkNotify benchmarks the performance of notifying a subscription.
func BenchmarkNotify(b *testing.B) {
	id := ID("test")
	notifier := &Notifier{
		h:         &handler{conn: &mockConn{json.NewEncoder(io.Discard)}},
		sub:       &Subscription{ID: id},
		activated: true,
	}
	msg := &types.Header{
		ParentHash: common.HexToHash("0x01"),
		Number:     big.NewInt(100),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		notifier.Notify(id, msg)
	}
}

func TestNotify(t *testing.T) {
	out := new(bytes.Buffer)
	id := ID("test")
	notifier := &Notifier{
		h:         &handler{conn: &mockConn{json.NewEncoder(out)}},
		sub:       &Subscription{ID: id},
		activated: true,
	}
	msg := &types.Header{
		ParentHash: common.HexToHash("0x01"),
		Number:     big.NewInt(100),
	}
	notifier.Notify(id, msg)
	have := strings.TrimSpace(out.String())
	want := `{"jsonrpc":"2.0","method":"_subscription","params":{"subscription":"test","result":{"parentHash":"0x0000000000000000000000000000000000000000000000000000000000000001","sha3Uncles":"0x0000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","transactionsRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","receiptsRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","difficulty":null,"number":"0x64","gasLimit":"0x0","gasUsed":"0x0","timestamp":"0x0","extraData":"0x","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","baseFeePerGas":null,"withdrawalsRoot":null,"blobGasUsed":null,"excessBlobGas":null,"parentBeaconBlockRoot":null,"hash":"0xe5fb877dde471b45b9742bb4bb4b3d74a761e2fb7cb849a3d2b687eed90fb604"}}}`
	if have != want {
		t.Errorf("have:\n%v\nwant:\n%v\n", have, want)
	}
}
