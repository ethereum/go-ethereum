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

type NotificationTestService struct{}

var (
	unsubCallbackCalled = false
)

func (s *NotificationTestService) Unsubscribe(subid string) {
	unsubCallbackCalled = true
}

func (s *NotificationTestService) SomeSubscription(ctx context.Context, n, val int) (Subscription, error) {
	notifier, supported := NotifierFromContext(ctx)
	if !supported {
		return nil, ErrNotificationsUnsupported
	}

	// by explicitly creating an subscription we make sure that the subscription id is send back to the client
	// before the first subscription.Notify is called. Otherwise the events might be send before the response
	// for the eth_subscribe method.
	subscription, err := notifier.NewSubscription(s.Unsubscribe)
	if err != nil {
		return nil, err
	}

	go func() {
		for i := 0; i < n; i++ {
			if err := subscription.Notify(val + i); err != nil {
				return
			}
		}
	}()

	return subscription, nil
}

func TestNotifications(t *testing.T) {
	server := NewServer()
	service := &NotificationTestService{}

	if err := server.RegisterName("eth", service); err != nil {
		t.Fatalf("unable to register test service %v", err)
	}

	clientConn, serverConn := net.Pipe()

	go server.ServeCodec(NewJSONCodec(serverConn), OptionMethodInvocation|OptionSubscriptions)

	out := json.NewEncoder(clientConn)
	in := json.NewDecoder(clientConn)

	n := 5
	val := 12345
	request := map[string]interface{}{
		"id":      1,
		"method":  "eth_subscribe",
		"version": "2.0",
		"params":  []interface{}{"someSubscription", n, val},
	}

	// create subscription
	if err := out.Encode(request); err != nil {
		t.Fatal(err)
	}

	var subid string
	response := JSONSuccessResponse{Result: subid}
	if err := in.Decode(&response); err != nil {
		t.Fatal(err)
	}

	var ok bool
	if subid, ok = response.Result.(string); !ok {
		t.Fatalf("expected subscription id, got %T", response.Result)
	}

	for i := 0; i < n; i++ {
		var notification jsonNotification
		if err := in.Decode(&notification); err != nil {
			t.Fatalf("%v", err)
		}

		if int(notification.Params.Result.(float64)) != val+i {
			t.Fatalf("expected %d, got %d", val+i, notification.Params.Result)
		}
	}

	clientConn.Close() // causes notification unsubscribe callback to be called
	time.Sleep(1 * time.Second)

	if !unsubCallbackCalled {
		t.Error("unsubscribe callback not called after closing connection")
	}
}
