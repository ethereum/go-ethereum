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
	"net/http"

	"github.com/gorilla/websocket"
)

// ClientOption is a configuration option for the RPC client.
type ClientOption interface {
	applyOption(*clientConfig)
}

type clientConfig struct {
	// HTTP settings
	httpClient  *http.Client
	httpHeaders http.Header
	httpAuth    HTTPAuth

	// WebSocket options
	wsDialer           *websocket.Dialer
	wsMessageSizeLimit *int64 // wsMessageSizeLimit nil = default, 0 = no limit

	// RPC handler options
	idgen              func() ID
	batchItemLimit     int
	batchResponseLimit int

	// Interceptors
	requestInterceptors  []RequestInterceptor
	responseInterceptors []ResponseInterceptor
}

func (cfg *clientConfig) initHeaders() {
	if cfg.httpHeaders == nil {
		cfg.httpHeaders = make(http.Header)
	}
}

func (cfg *clientConfig) setHeader(key, value string) {
	cfg.initHeaders()
	cfg.httpHeaders.Set(key, value)
}

type optionFunc func(*clientConfig)

func (fn optionFunc) applyOption(opt *clientConfig) {
	fn(opt)
}

// WithWebsocketDialer configures the websocket.Dialer used by the RPC client.
func WithWebsocketDialer(dialer websocket.Dialer) ClientOption {
	return optionFunc(func(cfg *clientConfig) {
		cfg.wsDialer = &dialer
	})
}

// WithWebsocketMessageSizeLimit configures the websocket message size limit used by the RPC
// client. Passing a limit of 0 means no limit.
func WithWebsocketMessageSizeLimit(messageSizeLimit int64) ClientOption {
	return optionFunc(func(cfg *clientConfig) {
		cfg.wsMessageSizeLimit = &messageSizeLimit
	})
}

// WithHeader configures HTTP headers set by the RPC client. Headers set using this option
// will be used for both HTTP and WebSocket connections.
func WithHeader(key, value string) ClientOption {
	return optionFunc(func(cfg *clientConfig) {
		cfg.initHeaders()
		cfg.httpHeaders.Set(key, value)
	})
}

// WithHeaders configures HTTP headers set by the RPC client. Headers set using this
// option will be used for both HTTP and WebSocket connections.
func WithHeaders(headers http.Header) ClientOption {
	return optionFunc(func(cfg *clientConfig) {
		cfg.initHeaders()
		for k, vs := range headers {
			cfg.httpHeaders[k] = vs
		}
	})
}

// WithHTTPClient configures the http.Client used by the RPC client.
func WithHTTPClient(c *http.Client) ClientOption {
	return optionFunc(func(cfg *clientConfig) {
		cfg.httpClient = c
	})
}

// WithHTTPAuth configures HTTP request authentication. The given provider will be called
// whenever a request is made. Note that only one authentication provider can be active at
// any time.
func WithHTTPAuth(a HTTPAuth) ClientOption {
	if a == nil {
		panic("nil auth")
	}
	return optionFunc(func(cfg *clientConfig) {
		cfg.httpAuth = a
	})
}

// A HTTPAuth function is called by the client whenever a HTTP request is sent.
// The function must be safe for concurrent use.
//
// Usually, HTTPAuth functions will call h.Set("authorization", "...") to add
// auth information to the request.
type HTTPAuth func(h http.Header) error

// WithBatchItemLimit changes the maximum number of items allowed in batch requests.
//
// Note: this option applies when processing incoming batch requests. It does not affect
// batch requests sent by the client.
func WithBatchItemLimit(limit int) ClientOption {
	return optionFunc(func(cfg *clientConfig) {
		cfg.batchItemLimit = limit
	})
}

// WithBatchResponseSizeLimit changes the maximum number of response bytes that can be
// generated for batch requests. When this limit is reached, further calls in the batch
// will not be processed.
//
// Note: this option applies when processing incoming batch requests. It does not affect
// batch requests sent by the client.
func WithBatchResponseSizeLimit(sizeLimit int) ClientOption {
	return optionFunc(func(cfg *clientConfig) {
		cfg.batchResponseLimit = sizeLimit
	})
}

// RequestInterceptor is called before sending RPC requests.
//
// The interceptor is invoked with the request context, method name, and arguments.
// For batch requests, method is empty string and args is nil; the interceptor runs
// once per batch, not per item.
//
// Request interceptors run in order. If an interceptor returns an error, the request
// is not sent and the error is returned to the caller immediately.
//
// The context passed to the interceptor is the same context passed to CallContext.
// Interceptors can use the context for rate limiting (e.g., limiter.Wait(ctx)) or
// checking cancellation.
//
// IMPORTANT: Interceptors MUST NOT modify the args slice. Doing so results in
// undefined behavior and may break retries or reconnections.
type RequestInterceptor func(ctx context.Context, method string, args []interface{}) error

// ResponseInterceptor is called after receiving RPC responses.
//
// The interceptor is invoked with the request context, method name, and the final error
// (which may be nil on success, or an I/O error, RPC error, or unmarshal error).
//
// For batch requests, method is empty string and the interceptor runs once per batch.
// The error represents the transport-level error (usually nil if the batch request
// succeeded). Per-item RPC errors within the batch are not passed to interceptors;
// they remain in BatchElem.Error and should be checked by the caller.
//
// Response interceptors run in order. Each interceptor receives the error returned by
// the previous interceptor (or the original error for the first interceptor).
// The error returned by the last interceptor is returned to the caller.
//
// Interceptors can suppress errors by returning nil, wrap errors for additional context,
// or return a different error entirely.
type ResponseInterceptor func(ctx context.Context, method string, err error) error

// WithRequestInterceptor adds a request interceptor to the client.
//
// Request interceptors are called before sending RPC requests. Multiple interceptors
// can be added and will run in the order they were added. If any interceptor returns
// an error, the request is not sent.
//
// Example - rate limiting:
//
//	limiter := rate.NewLimiter(rate.Every(time.Second), 10)
//	client, _ := rpc.DialOptions(ctx, url,
//	    rpc.WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
//	        return limiter.Wait(ctx)
//	    }),
//	)
//
// Example - logging:
//
//	client, _ := rpc.DialOptions(ctx, url,
//	    rpc.WithRequestInterceptor(func(ctx context.Context, method string, args []interface{}) error {
//	        log.Printf("RPC call: %s", method)
//	        return nil
//	    }),
//	)
func WithRequestInterceptor(interceptor RequestInterceptor) ClientOption {
	return optionFunc(func(cfg *clientConfig) {
		cfg.requestInterceptors = append(cfg.requestInterceptors, interceptor)
	})
}

// WithResponseInterceptor adds a response interceptor to the client.
//
// Response interceptors are called after receiving RPC responses. Multiple interceptors
// can be added and will run in the order they were added. Each interceptor receives
// the error from the previous interceptor.
//
// Example - error logging:
//
//	client, _ := rpc.DialOptions(ctx, url,
//	    rpc.WithResponseInterceptor(func(ctx context.Context, method string, err error) error {
//	        if err != nil {
//	            log.Printf("RPC error for %s: %v", method, err)
//	        }
//	        return err
//	    }),
//	)
//
// For batch requests, if you need per-item error observability, check BatchElem.Error
// after the call returns:
//
//	batch := []rpc.BatchElem{...}
//	err := client.BatchCallContext(ctx, batch)
//	for i, elem := range batch {
//	    if elem.Error != nil {
//	        log.Printf("Batch[%d] %s failed: %v", i, elem.Method, elem.Error)
//	    }
//	}
func WithResponseInterceptor(interceptor ResponseInterceptor) ClientOption {
	return optionFunc(func(cfg *clientConfig) {
		cfg.responseInterceptors = append(cfg.responseInterceptors, interceptor)
	})
}
