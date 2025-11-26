# RPC Middleware Injection Strategies for Go-Ethereum

## Overview
This document outlines practical strategies for implementing middleware in the go-ethereum RPC client, considering the current architecture and limitations.

## Strategy 1: Wrapper Client (Recommended for Application-Level)

### Approach
Wrap the native Client with a custom struct that intercepts all method calls.

### Advantages
- Non-invasive (doesn't modify go-ethereum code)
- Works for all transports equally
- Can add logging, metrics, request/response transformation
- Easy to test and compose multiple middlewares

### Implementation Pattern
```go
type MiddlewareClient struct {
    client *rpc.Client
    middlewares []Middleware
}

type Middleware interface {
    BeforeCall(ctx context.Context, method string, args ...interface{}) error
    AfterCall(ctx context.Context, method string, result interface{}, err error) error
    OnSubscription(ctx context.Context, namespace string, channel interface{}) error
}

func (mc *MiddlewareClient) CallContext(ctx context.Context, result interface{}, 
    method string, args ...interface{}) error {
    
    // Before hooks
    for _, m := range mc.middlewares {
        if err := m.BeforeCall(ctx, method, args...); err != nil {
            return err
        }
    }
    
    // Call
    err := mc.client.CallContext(ctx, result, method, args...)
    
    // After hooks
    for _, m := range mc.middlewares {
        if hookErr := m.AfterCall(ctx, method, result, err); hookErr != nil {
            if err == nil {
                err = hookErr
            }
        }
    }
    
    return err
}
```

### Use Cases
- Request logging/tracing
- Retry logic
- Rate limiting
- Authentication token refresh
- Request/response transformation

---

## Strategy 2: HTTP-Specific Middleware (Best for HTTP Transport)

### Approach
Use http.RoundTripper wrapping when creating HTTP client.

### Advantages
- Transparent to go-ethereum code
- Full control over HTTP layer
- Can intercept headers, status codes, body
- Leverage standard Go HTTP middleware ecosystem

### Implementation Pattern
```go
type RoundTripperMiddleware struct {
    next http.RoundTripper
    middlewares []HTTPMiddleware
}

type HTTPMiddleware interface {
    BeforeRequest(req *http.Request) error
    AfterResponse(resp *http.Response) error
}

func (rtm *RoundTripperMiddleware) RoundTrip(req *http.Request) (*http.Response, error) {
    // Before hooks
    for _, m := range rtm.middlewares {
        if err := m.BeforeRequest(req); err != nil {
            return nil, err
        }
    }
    
    // Call
    resp, err := rtm.next.RoundTrip(req)
    
    // After hooks
    if resp != nil {
        for _, m := range rtm.middlewares {
            if hookErr := m.AfterResponse(resp); hookErr != nil {
                if err == nil {
                    err = hookErr
                }
            }
        }
    }
    
    return resp, err
}

// Usage
func NewHTTPClientWithMiddleware(middlewares ...HTTPMiddleware) *http.Client {
    base := &http.Client{}
    rt := &RoundTripperMiddleware{
        next: base.Transport,
        middlewares: middlewares,
    }
    base.Transport = rt
    return base
}

// Create RPC client with middleware
httpClient := NewHTTPClientWithMiddleware(
    &LoggingMiddleware{},
    &RetryMiddleware{},
)
rpcClient, _ := rpc.DialOptions(ctx, "http://localhost:8545",
    rpc.WithHTTPClient(httpClient),
)
```

### Use Cases
- HTTP-specific logging
- Response time measurement
- Status code handling
- Header inspection/modification
- Cookie handling
- Compression handling

---

## Strategy 3: HTTPAuth Hook (Built-in, Limited)

### Approach
Use existing WithHTTPAuth option to add authentication headers and basic logging.

### Advantages
- Built into go-ethereum
- No additional dependencies
- Applied per-request

### Limitations
- Only for HTTP
- Only manipulates headers
- No response interception
- No error handling

### Implementation Pattern
```go
type AuthMiddleware struct {
    token string
    logger Logger
}

func (am *AuthMiddleware) Authenticate(h http.Header) error {
    // Log the request
    am.logger.Debug("auth middleware: adding token")
    
    // Add auth header
    h.Set("Authorization", "Bearer " + am.token)
    return nil
}

// Usage
rpcClient, _ := rpc.DialOptions(ctx, "http://localhost:8545",
    rpc.WithHTTPAuth(authMiddleware.Authenticate),
)
```

### Use Cases
- Token/API key injection
- Basic auth setup
- Header logging

---

## Strategy 4: Context-Based Header Injection

### Approach
Use NewContextWithHeaders to inject per-request headers without modifying client config.

### Advantages
- Per-request granularity
- No global state
- Works with existing client
- Can be combined with other approaches

### Implementation Pattern
```go
func CallWithTraceID(client *rpc.Client, ctx context.Context, 
    result interface{}, method string, args ...interface{}) error {
    
    traceID := generateTraceID()
    headers := http.Header{
        "X-Trace-ID": []string{traceID},
        "X-Request-ID": []string{generateRequestID()},
    }
    
    ctx = rpc.NewContextWithHeaders(ctx, headers)
    return client.CallContext(ctx, result, method, args...)
}
```

### Use Cases
- Request ID/Trace ID injection
- Per-request metadata
- Dynamic header injection
- Request correlation

---

## Strategy 5: Message-Level Interception (Advanced)

### Approach
Wrap ServerCodec interface to intercept messages at codec level.

### Advantages
- Transport-agnostic (works for HTTP, WS, IPC)
- Full access to jsonrpcMessage
- Can transform requests/responses
- Enables comprehensive logging

### Challenges
- Requires deeper integration
- Complex state management
- Need to handle all codec types
- May affect performance

### Implementation Pattern
```go
// Wrapper for any ServerCodec
type InterceptingCodec struct {
    codec      rpc.ServerCodec
    interceptor MessageInterceptor
}

type MessageInterceptor interface {
    OnReadMessage(msg *jsonrpcMessage) error
    OnWriteMessage(msg *jsonrpcMessage) error
}

func (ic *InterceptingCodec) readBatch() ([]*jsonrpcMessage, bool, error) {
    msgs, batch, err := ic.codec.readBatch()
    if err == nil && ic.interceptor != nil {
        for _, msg := range msgs {
            if ierr := ic.interceptor.OnReadMessage(msg); ierr != nil {
                return nil, false, ierr
            }
        }
    }
    return msgs, batch, err
}

func (ic *InterceptingCodec) writeJSON(ctx context.Context, v interface{}, isError bool) error {
    // Would need to intercept at this level
    if msg, ok := v.(*jsonrpcMessage); ok && ic.interceptor != nil {
        if ierr := ic.interceptor.OnWriteMessage(msg); ierr != nil {
            return ierr
        }
    }
    return ic.codec.writeJSON(ctx, v, isError)
}

// Would need to implement other ServerCodec methods...
```

### Use Cases
- Request/response logging with full message body
- Message transformation/validation
- Performance metrics
- Rate limiting at RPC level

---

## Strategy 6: Dispatch Channel Interception (Advanced)

### Approach
Intercept at the channel layer in the Client's dispatch loop.

### Advantages
- Access to internal state (request IDs, handlers)
- Can correlate requests and responses
- Pure Go concurrency primitives

### Challenges
- Very tightly coupled to implementation
- Breaks encapsulation
- Complex to implement correctly
- Difficult to maintain across versions

### Not Recommended
This approach is too invasive and fragile. Prefer Strategies 1-5.

---

## Strategy 7: WebSocket-Specific Handlers

### Approach
For WebSocket connections, create wrapper around websocket.Dialer to customize connection behavior.

### Advantages
- WS-specific features possible
- Connection-level control
- Can inspect handshake

### Implementation Pattern
```go
type DialerWithMiddleware struct {
    base *websocket.Dialer
    middlewares []DialerMiddleware
}

type DialerMiddleware interface {
    BeforeDial(ctx context.Context, url string) error
    AfterDial(conn *websocket.Conn) error
}

func (dwm *DialerWithMiddleware) DialContext(ctx context.Context, 
    urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
    
    // Before hooks
    for _, m := range dwm.middlewares {
        if err := m.BeforeDial(ctx, urlStr); err != nil {
            return nil, nil, err
        }
    }
    
    // Dial
    conn, resp, err := dwm.base.DialContext(ctx, urlStr, requestHeader)
    
    // After hooks
    if err == nil && conn != nil {
        for _, m := range dwm.middlewares {
            if hookErr := m.AfterDial(conn); hookErr != nil {
                conn.Close()
                return nil, resp, hookErr
            }
        }
    }
    
    return conn, resp, err
}

// Usage
dialer := &DialerWithMiddleware{
    base: &websocket.Dialer{...},
    middlewares: []DialerMiddleware{
        &WSLoggingMiddleware{},
    },
}

rpcClient, _ := rpc.DialOptions(ctx, "ws://localhost:8545",
    rpc.WithWebsocketDialer(*dialer.base), // Note: can't pass wrapper directly
)
```

### Limitation
WithWebsocketDialer expects a websocket.Dialer directly, not a wrapper, so this approach has limited applicability without modifying client_opt.go.

---

## Strategy 8: Subscription Wrapper

### Approach
Wrap the ClientSubscription returned from Subscribe to intercept events.

### Advantages
- Subscription-specific handling
- Non-invasive
- Works with existing client

### Implementation Pattern
```go
type SubscriptionMiddleware struct {
    sub *rpc.ClientSubscription
    ch interface{}
    middlewares []SubscriptionMiddleware
}

type SubscriptionMiddleware interface {
    OnEvent(ev interface{}) error
    OnError(err error) error
}

func WrapSubscription(sub *rpc.ClientSubscription, 
    ch interface{}, middlewares ...SubscriptionMiddleware) *SubscriptionMiddleware {
    
    return &SubscriptionMiddleware{
        sub: sub,
        ch: ch,
        middlewares: middlewares,
    }
}

// Would need to read from sub.C and apply middlewares
```

---

## Recommended Strategy Selection

### For HTTP-Only Applications
1. Use Strategy 2 (HTTP RoundTripper) for transport-level middleware
2. Use Strategy 1 (Wrapper Client) for application-level logging/transformation
3. Use HTTPAuth for simple authentication

### For WebSocket Applications
1. Use Strategy 1 (Wrapper Client) for application-level concerns
2. Use Strategy 2 if also supporting HTTP
3. Use Context headers for per-request metadata

### For Comprehensive Tracing/Metrics
1. Combine Strategy 1 (Wrapper Client) with Strategy 2 (RoundTripper)
2. Use Strategy 4 (Context Headers) for correlation IDs
3. Avoid Strategy 5 unless absolutely necessary

### For Advanced Use Cases
1. Implement custom strategies based on application requirements
2. Consider whether modifying go-ethereum is acceptable for your use case
3. Always prefer non-invasive wrappers over modifying the library

---

## Implementation Checklist

When implementing middleware for go-ethereum RPC client:

- [ ] Identify which transports need to be supported (HTTP, WS, IPC, Stdio)
- [ ] Determine middleware scope (connection-level, request-level, message-level)
- [ ] Choose non-invasive approach when possible
- [ ] Handle context cancellation properly
- [ ] Implement error handling and propagation
- [ ] Consider performance impact (avoid allocations in hot path)
- [ ] Add tests for middleware behavior
- [ ] Document expected behavior and limitations
- [ ] Plan for go-ethereum version upgrades
- [ ] Consider thread safety for concurrent calls
- [ ] Handle subscription/notification middleware separately if needed
- [ ] Implement proper logging without noise

---

## Anti-Patterns to Avoid

1. **Blocking in Middleware**: Don't block indefinitely; respect context timeouts
2. **Global State**: Avoid global variables; use dependency injection
3. **Ignoring Errors**: Always propagate errors from hooks
4. **Transport Assumptions**: Don't assume HTTP if WS/IPC might be used
5. **Tight Coupling**: Don't depend on private go-ethereum fields
6. **Synchronous I/O**: Avoid synchronous network calls in hot path
7. **Memory Leaks**: Always clean up goroutines and channels
8. **Silent Failures**: Log all middleware errors, don't swallow them
