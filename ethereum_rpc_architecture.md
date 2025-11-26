# Go-Ethereum RPC Client Architecture Overview

## 1. Client Structure & Initialization

### Core Client struct (rpc/client.go)
The `Client` struct is the main entry point for RPC communication:

```go
type Client struct {
    idgen    func() ID              // subscription ID generator
    isHTTP   bool                   // connection type: http, ws, or ipc
    services *serviceRegistry       // service registry for method resolution
    
    idCounter atomic.Uint32         // counter for request IDs
    
    // Connection management
    reconnectFunc reconnectFunc      // function to establish new connections
    writeConn jsonWriter             // current connection (wrapped in httpConn, websocketCodec, or jsonCodec)
    
    // Dispatch system (for non-HTTP)
    close       chan struct{}        // signal to close client
    closing     chan struct{}        // closed when client is quitting
    didClose    chan struct{}        // closed when client quits
    reconnected chan ServerCodec     // where write/reconnect sends new connections
    readOp      chan readOp          // read messages from connection
    readErr     chan error           // errors from read loop
    reqInit     chan *requestOp      // register response IDs, takes write lock
    reqSent     chan error           // signals write completion, releases write lock
    reqTimeout  chan *requestOp      // removes response IDs when call timeout expires
    
    // Configuration
    batchItemLimit       int
    batchResponseMaxSize int
}
```

### Initialization Flow

1. **Dial** → **DialContext** → **DialOptions** (public API entry points)
2. **DialOptions** parses URL and creates appropriate transport:
   - HTTP/HTTPS → `newClientTransportHTTP()`
   - WS/WSS → `newClientTransportWS()`
   - IPC → `newClientTransportIPC()`
   - stdio → `newClientTransportIO()`

3. **newClient()** creates the Client and initializes dispatch loop:
   ```go
   func newClient(initctx context.Context, cfg *clientConfig, connect reconnectFunc) (*Client, error) {
       conn, err := connect(initctx)     // Establish initial connection
       if err != nil {
           return nil, err
       }
       c := initClient(conn, new(serviceRegistry), cfg)
       c.reconnectFunc = connect         // Store reconnection function
       return c, nil
   }
   ```

4. **initClient()** sets up the Client:
   - Creates channels for dispatch
   - Determines if HTTP or not (HTTP doesn't use dispatch loop)
   - Launches dispatch goroutine for non-HTTP connections

## 2. Configuration System (client_opt.go)

### ClientOption Pattern
Uses functional options pattern for flexible configuration:

```go
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
    wsMessageSizeLimit *int64
    
    // RPC handler options
    idgen              func() ID
    batchItemLimit     int
    batchResponseLimit int
}
```

### Available Options
- `WithHTTPClient(client)` - Custom HTTP client
- `WithHTTPAuth(authFunc)` - Authentication provider called per request
- `WithHeader(key, value)` - Custom HTTP headers
- `WithHeaders(header)` - Multiple headers
- `WithWebsocketDialer(dialer)` - Custom WS dialer
- `WithWebsocketMessageSizeLimit(limit)` - WS message size limit
- `WithBatchItemLimit(limit)` - Batch request limits
- `WithBatchResponseSizeLimit(limit)` - Batch response size limits

**Key Insight**: These options only configure the *client-side* creation. HTTPAuth is called during request preparation.

## 3. Connection Handling

### Three Main Transport Types

#### A. HTTP Transport (http.go)
- **httpConn struct**: Wrapper that implements ServerCodec interface (but mostly stubbed)
- **HTTP-specific behavior**:
  - No persistent connection (stateless)
  - No dispatch loop needed
  - Direct request/response cycle
  - Headers managed in `httpConn.headers` (protected by mutex)
  - Authentication via `HTTPAuth` function applied per-request

- **Request flow** (sendHTTP):
  ```
  Client.CallContext() 
    → Client.sendHTTP()          // Directly send via HTTP
      → httpConn.doRequest()     // Marshal, create HTTP request, auth, execute
        → http.Client.Do()       // Execute HTTP request
      → JSON decode response
      → op.resp <- response
  ```

#### B. WebSocket Transport (websocket.go)
- **websocketCodec struct**: Implements ServerCodec
- **Features**:
  - Persistent connection
  - Ping/pong keepalive (30s interval)
  - Message size limit (default 32MB)
  - Origin validation
  - Connection pooling for write buffers

- **Creation**: 
  ```go
  newClientTransportWS()
    → Create websocket.Dialer
    → Apply custom headers and auth
    → Return connect() function that:
        - Calls dialer.DialContext()
        - Wraps in newWebsocketCodec()
  ```

- **Ping loop**: Separate goroutine in websocketCodec keeps connection alive

#### C. IPC/Stdio Transport
- **jsonCodec struct**: Standard JSON codec wrapper
- Simpler than WS, used for local domain sockets and stdio

### ServerCodec Interface (types.go)
The abstraction that all transports must implement:

```go
type ServerCodec interface {
    peerInfo() PeerInfo              // Return connection metadata
    readBatch() (msgs, isBatch, err) // Read and parse JSON-RPC messages
    close()                          // Close the connection
    jsonWriter                       // Embedded interface
}

type jsonWriter interface {
    writeJSON(ctx context.Context, msg interface{}, isError bool) error
    closed() <-chan interface{}      // Channel closed when connection ends
    remoteAddr() string              // Peer address
}
```

## 4. RPC Call Flow & Request Handling

### Single Request Flow (CallContext)
```
Client.CallContext(ctx, result, method, args...)
  1. Validate result is pointer or nil
  2. Create jsonrpcMessage with:
     - Version: "2.0"
     - ID: Next ID from counter
     - Method: Requested method
     - Params: JSON-encoded arguments
  3. Create requestOp with:
     - IDs: [msg.ID]
     - resp: channel for responses (buffered)
     - err: any error
  4. IF HTTP:
       → Client.sendHTTP(ctx, op, msg)
          → httpConn.doRequest() sends HTTP POST
          → response decoded and sent to op.resp
     ELSE:
       → Client.send(ctx, op, msg)
          → Send op to reqInit channel (dispatch picks it up)
          → Send msg on connection via c.write()
          → Handler receives response, routes to op.resp
  5. op.wait(ctx, c) blocks until:
     - Context canceled (timeout)
     - Response received on op.resp
  6. Decode response and unmarshal into result
```

### Batch Request Flow (BatchCallContext)
Similar to single request but:
- Creates multiple jsonrpcMessage objects
- Sends all at once via sendBatchHTTP or send
- Maps response IDs back to original request elements
- Stores errors in BatchElem.Error fields

### Dispatch Loop (Non-HTTP Only)
The dispatch goroutine (`Client.dispatch()`) is the heart of non-HTTP clients:

```go
func (c *Client) dispatch(codec ServerCodec) {
    conn := c.newClientConn(codec)  // Create handler for this connection
    go c.read(codec)                 // Launch read loop
    
    for {
        select {
        // Close signal
        case <-c.close:
            return
        
        // Read path: incoming messages
        case op := <-c.readOp:        // Messages from read loop
            if op.batch:
                conn.handler.handleBatch(op.msgs)
            else:
                conn.handler.handleMsg(op.msgs[0])
        
        case err := <-c.readErr:      // Read error
            conn.close(err, lastOp)
            reading = false
        
        // Reconnect path: new connection
        case newcodec := <-c.reconnected:
            conn.close(errClientReconnected, lastOp)
            conn = c.newClientConn(newcodec)
            conn.handler.addRequestOp(lastOp)
        
        // Send path: outgoing requests
        case op := <-c.reqInit:       // New request to send
            reqInitLock = nil          // Take write lock
            conn.handler.addRequestOp(op)
        
        case err := <-c.reqSent:      // Send complete
            if err != nil:
                conn.handler.removeRequestOp(lastOp)
            reqInitLock = c.reqInit    // Release write lock
        
        // Timeout path
        case op := <-c.reqTimeout:
            conn.handler.removeRequestOp(op)
        }
    }
}
```

### Read Loop
```go
func (c *Client) read(codec ServerCodec) {
    for {
        msgs, batch, err := codec.readBatch()  // Block reading from connection
        if err != nil {
            c.readErr <- err
            return
        }
        c.readOp <- readOp{msgs, batch}  // Send to dispatch
    }
}
```

### Handler (handler.go)
The handler processes messages and manages subscriptions:
- Maps request IDs to pending requestOp objects
- Routes responses to waiting callers
- Manages subscriptions
- Handles timeouts
- Processes batches with response limits

## 5. WebSocket Connection Details

### WebSocket Dial (DialWebsocket / DialOptions with WS URL)
1. Parse endpoint URL
2. Extract origin and basic auth from URL
3. Apply custom headers and auth from config
4. Create websocket.Dialer with:
   - ReadBufferSize: 1024
   - WriteBufferSize: 1024
   - WriteBufferPool: Shared sync.Pool for efficiency
   - Proxy: http.ProxyFromEnvironment
5. DialContext with prepared headers
6. Wrap connection in websocketCodec
7. Codec starts pingLoop goroutine

### WebSocket Message Size
- Default read limit: 32 MB (wsDefaultReadLimit)
- Configurable via WithWebsocketMessageSizeLimit
- Connection reads with codec.SetReadLimit()

### WebSocket Ping/Pong
- Ping sent every 30s when idle
- Pong handler resets read deadline
- Write timeout for ping: 5s
- Pong expected within: 30s

### WebSocket Headers
- Origin header set (for CORS)
- User-Agent preserved
- Custom headers from config applied
- HTTP auth applied during connection

## 6. Context & Header Management

### HTTP Headers in Context (context_headers.go)
Headers can be injected via context for per-request customization:

```go
// Create context with headers
ctx := NewContextWithHeaders(context.Background(), headers)

// Called with HTTP client:
client.CallContext(ctx, result, "method")

// In doRequest(), headers are extracted and merged
func headersFromContext(ctx context.Context) http.Header
func setHeaders(dst http.Header, src http.Header) http.Header
```

**Important**: Headers from context are merged with static headers, context headers override.

### Client Context Extraction
Via `ClientFromContext(ctx)`:
- Returns the Client associated with a request context
- Used for "reverse calls" in handler methods
- Enables handler methods to call back out on the client

## 7. Middleware Injection Points (Currently Limited)

### Existing Extension Points

1. **HTTPAuth Function**
   - Called during every HTTP request
   - Has full access to request headers
   - Can add authentication headers
   - **Limitation**: Only on HTTP, doesn't apply to WS

2. **Custom HTTP Client**
   - Can implement http.RoundTripper wrapper
   - Can intercept all HTTP traffic
   - Applied at HTTP client level
   - **Limitation**: Only HTTP

3. **Custom WebSocket Dialer**
   - Can implement custom dialing logic
   - Called for initial connection + reconnects
   - Limited middleware capability

4. **HTTP Headers via Context**
   - Per-request header injection
   - Applied in doRequest()
   - Limited to header manipulation

### Missing Middleware Patterns

1. **No request/response interception for non-HTTP**
   - WebSocket, IPC, Stdio bypass all middleware
   - Direct ServerCodec interface prevents layering

2. **No request/response logging hook**
   - No way to intercept jsonrpcMessage before/after
   - No built-in tracing/metrics

3. **No error interception**
   - No hook to transform or log errors
   - No metrics collection

4. **No subscription interception**
   - Subscribe requests bypass middleware
   - Subscription messages not intercepted

5. **No connection-level hooks**
   - No way to inject before connection established
   - No way to hook connection failures

## 8. Critical Code Paths for Middleware

### HTTP Path (Most Middleware-Friendly)
```
Client.CallContext()
  → Client.sendHTTP()
    → httpConn.doRequest()
      1. json.Marshal(msg)              ← Can intercept request
      2. http.NewRequestWithContext()
      3. req.Header = hc.headers.Clone()
      4. setHeaders(req.Header, headersFromContext(ctx))  ← Can add headers
      5. if hc.auth != nil: hc.auth(req.Header)           ← HTTPAuth hook
      6. resp, err := hc.client.Do(req)  ← Standard HTTP transport
      7. json.Decoder(respBody).Decode(&resp)  ← Can intercept response
```

### Non-HTTP Path (Limited Middleware)
```
Client.send()
  → Client.write()
    → c.writeConn.writeJSON(ctx, msg, isError)
      → jsonCodec.writeJSON()
        → c.encode(v, isErrorResponse)  ← Direct function call
          
In parallel:
c.read()
  → codec.readBatch()
    → c.decode(&rawmsg)  ← Direct function call
```

## 9. Key Insights for Middleware Design

### 1. Transport Asymmetry
- HTTP has good middleware hooks (HTTPAuth, context headers, http.Client)
- WebSocket/IPC/Stdio have limited hooks (only custom dialer)
- Middleware needs transport-specific implementation

### 2. Channel-Based Architecture
- Non-HTTP uses Go channels for dispatch
- Messages flow through defined channels (readOp, readErr, reqInit, reqSent)
- Could intercept at channel boundaries

### 3. Two-Layer Codec System
- Transport layer (httpConn, websocketCodec, jsonCodec) - implements ServerCodec
- Handler layer (handler) - processes jsonrpcMessage structs
- Middleware could target either layer

### 4. Request ID Tracking
- Every request assigned unique ID
- Can correlate requests/responses
- Enables request tracing

### 5. Connection Lifecycle
- Connections can be replaced (reconnect)
- New handler created per connection (newClientConn)
- Connection metadata available (peerInfo)

### 6. Error Handling
- Transport errors: returned from send/write
- RPC errors: returned in jsonError in response
- Both should be intercepted separately

### 7. Subscription Complexity
- Subscriptions require persistent connection (not HTTP)
- Messages flowing to handler via readOp
- Notifier pattern for server-side pushes
- Need special handling for subscription responses

## 10. Architecture Summary

```
┌─────────────────────────────────────────────────────────────┐
│  Application Code                                            │
│  client.CallContext() / client.Subscribe() / etc.          │
└────────────────────────┬────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
         ▼               ▼               ▼
    ┌────────┐    ┌──────────┐    ┌──────────┐
    │  HTTP  │    │WebSocket │    │   IPC    │
    │Handler │    │ Codec    │    │  Codec   │
    └────┬───┘    └────┬─────┘    └────┬─────┘
         │             │               │
         ▼             ▼               ▼
    ┌────────┐    ┌──────────┐    ┌──────────┐
    │httpConn│    │websocket │    │ jsonCodec│
    │        │    │ Codec    │    │          │
    └────┬───┘    └────┬─────┘    └────┬─────┘
         │             │               │
    [sendHTTP] [Dispatch Loop]  [Dispatch Loop]
         │             │               │
         ▼             ▼               ▼
    [HTTP Req]  [Channel Send]  [Channel Send]
         │             │               │
         │             └─────┬─────────┘
         │                   │
         └───────────┬───────┘
                     │
              [Network I/O]
                     │
         ┌───────────┴────────────┐
         ▼                        ▼
    [RPC Server]           [Other Clients]
```

This architecture shows that middleware injection points exist at:
- Application layer (wrapping Client)
- HTTP layer (custom client, headers, auth)
- Transport layer (custom dialer for WS)
- Handler layer (if we extend handler)
- Channel layer (if we intercept dispatch channels)
