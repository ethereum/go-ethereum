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
	"net"
	"testing"
	"time"

	"golang.org/x/net/context"
)

type CancelTestService struct{}

func (s *CancelTestService) BlockingFunction(ctx context.Context) error {
	select {
	case <-time.After(time.Second):
		return nil
	case <-ctx.Done():
		time.Sleep(time.Millisecond * 200)
		return ctx.Err()
	}
}

func TestCancel(t *testing.T) {
	server := NewServer()
	service := &CancelTestService{}

	if err := server.RegisterName("eth", service); err != nil {
		t.Fatalf("unable to register test service %v", err)
	}

	clientConn, serverConn := net.Pipe()

	go server.ServeCodec(NewJSONCodec(serverConn), OptionMethodInvocation|OptionSubscriptions)

	out := json.NewEncoder(clientConn)
	in := json.NewDecoder(clientConn)

	request := map[string]interface{}{
		"id":      1,
		"method":  "eth_blockingFunction",
		"version": "2.0",
		"params":  []interface{}{},
	}

	cancelRequest := map[string]interface{}{
		"id":      2,
		"method":  "rpc_cancel",
		"version": "2.0",
		"params":  []interface{}{1},
	}

	// test uncanceled request
	start := time.Now()
	// send request
	if err := out.Encode(request); err != nil {
		t.Fatal(err)
	}

	var response JSONErrResponse
	if err := in.Decode(&response); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	msg := response.Error.Message
	// expect uncanceled request to return after 1000ms with no error
	if elapsed < time.Millisecond*900 {
		t.Errorf("uncanceled request returned too early (elapsed: %v, expected: 1s)", elapsed)
	}
	if msg != "" {
		t.Errorf("uncanceled request returned with unexpected error: %v", msg)
	}
	time.Sleep(time.Millisecond * 10)

	// test canceled request
	start = time.Now()
	// send request again
	if err := out.Encode(request); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 500)
	// send cancel request
	if err := out.Encode(cancelRequest); err != nil {
		t.Fatal(err)
	}

	if err := in.Decode(&response); err != nil {
		t.Fatal(err)
	}
	elapsed = time.Since(start)
	msg = response.Error.Message
	// expect cancel request to return after 500ms with no error
	if elapsed > time.Millisecond*600 {
		t.Errorf("cancel request returned too late (elapsed: %v, expected: 500ms)", elapsed)
	}
	if msg != "" {
		t.Errorf("cancel request returned with unexpected error: %v", msg)
	}

	if err := in.Decode(&response); err != nil {
		t.Fatal(err)
	}
	elapsed = time.Since(start)
	msg = response.Error.Message
	// expect canceled request to return after 700ms with "context canceled" error
	if elapsed < time.Millisecond*600 {
		t.Errorf("canceled request returned too early (elapsed: %v, expected: 700ms)", elapsed)
	}
	if elapsed > time.Millisecond*800 {
		t.Errorf("canceled request returned too late (elapsed: %v, expected: 700ms)", elapsed)
	}
	if msg != "context canceled" {
		t.Errorf("canceled request returned with unexpected or no error: \"%v\" (expected: \"context canceled\")", msg)
	}

	clientConn.Close()
}

func TestPending(t *testing.T) {
	server := NewServer()
	service := &CancelTestService{}

	if err := server.RegisterName("eth", service); err != nil {
		t.Fatalf("unable to register test service %v", err)
	}

	clientConn, serverConn := net.Pipe()

	go server.ServeCodec(NewJSONCodec(serverConn), OptionMethodInvocation|OptionSubscriptions)

	out := json.NewEncoder(clientConn)
	in := json.NewDecoder(clientConn)

	request := map[string]interface{}{
		"id":      1,
		"method":  "eth_blockingFunction",
		"version": "2.0",
		"params":  []interface{}{},
	}

	cancelRequest := map[string]interface{}{
		"id":      123,
		"method":  "rpc_cancel",
		"version": "2.0",
		"params":  []interface{}{1},
	}

	pendingRequest := map[string]interface{}{
		"id":      456,
		"method":  "rpc_pending",
		"version": "2.0",
		"params":  []interface{}{},
	}

	ids := []interface{}{1, "qwerty", 2, 42}
	for _, id := range ids {
		request["id"] = id
		if err := out.Encode(request); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(time.Millisecond * 10)

	testPending := func(expRunning, expCanceled int) {
		if err := out.Encode(pendingRequest); err != nil {
			t.Fatal(err)
		}
		var response struct {
			Version string              `json:"jsonrpc"`
			Id      interface{}         `json:"id,omitempty"`
			Result  []jsonPendingStatus `json:"result"`
		}

		if err := in.Decode(&response); err != nil {
			t.Fatal(err)
		}
		p := response.Result
		r := 0
		c := 0
		for _, status := range p {
			if status.Status == "running" {
				r++
			} else {
				if status.Status == "canceled" {
					c++
				} else {
					t.Errorf("unknown pending request status: %s", status.Status)
				}
			}
		}
		if r != expRunning || c != expCanceled {
			t.Errorf("incorrect pending status results (got %d running / %d canceled, expected %d running / %d canceled)", r, c, expRunning, expCanceled)
		}
		time.Sleep(time.Millisecond * 10)
	}

	cancel := func(id interface{}, expMsg string) {
		// send cancel request
		cancelRequest["params"] = []interface{}{id}
		if err := out.Encode(cancelRequest); err != nil {
			t.Fatal(err)
		}
		var response JSONErrResponse
		if err := in.Decode(&response); err != nil {
			t.Fatal(err)
		}
		msg := response.Error.Message
		if msg != expMsg {
			t.Errorf("wrong error message from cancel (got \"%s\", expected \"%s\")", msg, expMsg)
		}
		time.Sleep(time.Millisecond * 10)
	}

	wait := func(count int) {
		for i := 0; i < count; i++ {
			var response JSONErrResponse
			if err := in.Decode(&response); err != nil {
				t.Fatal(err)
			}
			msg := response.Error.Message
			expMsg := "context canceled"
			if msg != expMsg {
				t.Errorf("wrong error message from canceled request (got \"%s\", expected \"%s\")", msg, expMsg)
			}
		}
		time.Sleep(time.Millisecond * 10)
	}

	testPending(4, 0)
	cancel(ids[0], "")
	testPending(3, 1)
	cancel(ids[1], "")
	testPending(2, 2)
	cancel("twrtret", "pending request not found")
	testPending(2, 2)
	wait(2)
	testPending(2, 0)
	cancel("", "")
	testPending(0, 2)
	wait(2)
	testPending(0, 0)

	clientConn.Close()
}
