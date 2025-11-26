// Copyright 2022 The go-ethereum Authors
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
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestInterceptor(t *testing.T) {
	// Setup a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprintln(w, `{"jsonrpc":"2.0","id":1,"result":"0x1"}`)
	}))
	defer server.Close()

	// Test that request interceptor is called
	var called bool
	var capturedMethod string
	client, err := DialOptions(context.Background(), server.URL,
		WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
			called = true
			capturedMethod = method
			return nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result string
	err = client.CallContext(context.Background(), &result, "test_method")
	if err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Error("request interceptor was not called")
	}
	if capturedMethod != "test_method" {
		t.Errorf("interceptor got method %q, want %q", capturedMethod, "test_method")
	}
}

func TestRequestInterceptorBlocks(t *testing.T) {
	// Setup a test server that should never be hit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not have been called")
	}))
	defer server.Close()

	// Test that request interceptor can block the request
	blockErr := errors.New("blocked by interceptor")
	client, err := DialOptions(context.Background(), server.URL,
		WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
			return blockErr
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result string
	err = client.CallContext(context.Background(), &result, "test_method")
	if err != blockErr {
		t.Errorf("got error %v, want %v", err, blockErr)
	}
}

func TestRequestInterceptorChaining(t *testing.T) {
	// Setup a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprintln(w, `{"jsonrpc":"2.0","id":1,"result":"0x1"}`)
	}))
	defer server.Close()

	// Test that multiple interceptors run in order
	var order []int
	client, err := DialOptions(context.Background(), server.URL,
		WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
			order = append(order, 1)
			return nil
		}),
		WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
			order = append(order, 2)
			return nil
		}),
		WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
			order = append(order, 3)
			return nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result string
	err = client.CallContext(context.Background(), &result, "test_method")
	if err != nil {
		t.Fatal(err)
	}

	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("interceptors ran in wrong order: %v", order)
	}
}

func TestRequestInterceptorShortCircuit(t *testing.T) {
	// Setup a test server that should never be hit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not have been called")
	}))
	defer server.Close()

	// Test that first error stops the chain
	blockErr := errors.New("blocked")
	var thirdCalled bool
	client, err := DialOptions(context.Background(), server.URL,
		WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
			return nil
		}),
		WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
			return blockErr
		}),
		WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
			thirdCalled = true
			return nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result string
	err = client.CallContext(context.Background(), &result, "test_method")
	if err != blockErr {
		t.Errorf("got error %v, want %v", err, blockErr)
	}
	if thirdCalled {
		t.Error("third interceptor should not have been called")
	}
}

func TestResponseInterceptor(t *testing.T) {
	// Setup a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprintln(w, `{"jsonrpc":"2.0","id":1,"result":"0x1"}`)
	}))
	defer server.Close()

	// Test that response interceptor is called with nil error on success
	var called bool
	var capturedMethod string
	var capturedErr error
	client, err := DialOptions(context.Background(), server.URL,
		WithResponseInterceptor(func(ctx context.Context, method string, err error) error {
			called = true
			capturedMethod = method
			capturedErr = err
			return err
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result string
	err = client.CallContext(context.Background(), &result, "test_method")
	if err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Error("response interceptor was not called")
	}
	if capturedMethod != "test_method" {
		t.Errorf("interceptor got method %q, want %q", capturedMethod, "test_method")
	}
	if capturedErr != nil {
		t.Errorf("interceptor got error %v, want nil", capturedErr)
	}
}

func TestResponseInterceptorWithError(t *testing.T) {
	// Setup a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprintln(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"test error"}}`)
	}))
	defer server.Close()

	// Test that response interceptor receives the error
	var capturedErr error
	client, err := DialOptions(context.Background(), server.URL,
		WithResponseInterceptor(func(ctx context.Context, method string, err error) error {
			capturedErr = err
			return err
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result string
	err = client.CallContext(context.Background(), &result, "test_method")
	if err == nil {
		t.Fatal("expected error")
	}

	if capturedErr == nil {
		t.Error("interceptor should have received error")
	}
	if capturedErr.Error() != "test error" {
		t.Errorf("interceptor got error %q, want %q", capturedErr.Error(), "test error")
	}
}

func TestResponseInterceptorCanModifyError(t *testing.T) {
	// Setup a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprintln(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"original error"}}`)
	}))
	defer server.Close()

	// Test that response interceptor can wrap the error
	wrappedErr := errors.New("wrapped error")
	client, err := DialOptions(context.Background(), server.URL,
		WithResponseInterceptor(func(ctx context.Context, method string, err error) error {
			if err != nil {
				return wrappedErr
			}
			return err
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result string
	err = client.CallContext(context.Background(), &result, "test_method")
	if err != wrappedErr {
		t.Errorf("got error %v, want %v", err, wrappedErr)
	}
}

func TestResponseInterceptorChaining(t *testing.T) {
	// Setup a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprintln(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"original"}}`)
	}))
	defer server.Close()

	// Test that multiple response interceptors run in order and chain errors
	client, err := DialOptions(context.Background(), server.URL,
		WithResponseInterceptor(func(ctx context.Context, method string, err error) error {
			if err != nil {
				return fmt.Errorf("first: %w", err)
			}
			return err
		}),
		WithResponseInterceptor(func(ctx context.Context, method string, err error) error {
			if err != nil {
				return fmt.Errorf("second: %w", err)
			}
			return err
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result string
	err = client.CallContext(context.Background(), &result, "test_method")
	if err == nil {
		t.Fatal("expected error")
	}

	// Check that error was wrapped by both interceptors
	errMsg := err.Error()
	if errMsg != "second: first: original" {
		t.Errorf("got error %q, expected chained wrapping", errMsg)
	}
}

func TestBatchCallWithInterceptors(t *testing.T) {
	// Setup a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		fmt.Fprintln(w, `[{"jsonrpc":"2.0","id":1,"result":"0x1"},{"jsonrpc":"2.0","id":2,"result":"0x2"}]`)
	}))
	defer server.Close()

	// Test that interceptors are called for batch requests
	var reqCalled, respCalled bool
	var reqMethod, respMethod string
	client, err := DialOptions(context.Background(), server.URL,
		WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
			reqCalled = true
			reqMethod = method
			return nil
		}),
		WithResponseInterceptor(func(ctx context.Context, method string, err error) error {
			respCalled = true
			respMethod = method
			return err
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	batch := []BatchElem{
		{Method: "test_method1", Args: []interface{}{}, Result: new(string)},
		{Method: "test_method2", Args: []interface{}{}, Result: new(string)},
	}
	err = client.BatchCallContext(context.Background(), batch)
	if err != nil {
		t.Fatal(err)
	}

	if !reqCalled {
		t.Error("request interceptor was not called for batch")
	}
	if !respCalled {
		t.Error("response interceptor was not called for batch")
	}
	// For batch calls, method should be empty string
	if reqMethod != "" {
		t.Errorf("request interceptor got method %q, want empty string for batch", reqMethod)
	}
	if respMethod != "" {
		t.Errorf("response interceptor got method %q, want empty string for batch", respMethod)
	}
}

func TestNotifyWithInterceptors(t *testing.T) {
	// Test that request interceptor can block notifications.
	// We don't actually send the notification since Notify is primarily
	// for persistent connections (WebSocket/IPC), not HTTP.
	blockErr := errors.New("blocked notification")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not have been called")
	}))
	defer server.Close()

	client, err := DialOptions(context.Background(), server.URL,
		WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
			if method == "test_notification" {
				return blockErr
			}
			return nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	err = client.Notify(context.Background(), "test_notification")
	if err != blockErr {
		t.Errorf("got error %v, want %v", err, blockErr)
	}
}

func TestSubscribeWithInterceptors(t *testing.T) {
	// Test that request interceptor can block subscription requests.
	blockErr := errors.New("blocked subscribe")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not have been called")
	}))
	defer server.Close()

	client, err := DialOptions(context.Background(), server.URL,
		WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
			if method == "eth_subscribe" {
				return blockErr
			}
			return nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ch := make(chan interface{})
	_, err = client.EthSubscribe(context.Background(), ch, "newHeads")

	// Should get ErrNotificationsUnsupported for HTTP client first,
	// but if we had a WS client, the interceptor would block it.
	// For now, just verify HTTP correctly returns unsupported.
	if err != ErrNotificationsUnsupported {
		t.Errorf("got error %v, want %v", err, ErrNotificationsUnsupported)
	}
}
