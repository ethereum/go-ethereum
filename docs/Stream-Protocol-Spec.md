
stream! protocol
======

| Subject | Description |
|---|---|
| Authors | @zelig, @acud, @nonsense |
| Status | Draft |
| Created |	2019-06-11 |


### definition of stream
a protocol that facilitates data transmission between two swarm nodes, specifically targeting sequential data in the form of a sequence of chunks as defined by swarm. the protocol should cater for the following requirements: 
- client should be able to request arbitrary ranges from the server
- client can be assumed to have some of the data already and therefore can opt in to selectivally request chunks based on their hashes

As mentioned, the client is typically expected to have some of the data in the stream. to mitigate duplicate data transmission the stream protocol provides a configurable message roundtrip before batch delivery which allows the downstream peer to selectively request the chunks which it does not store at the time of the request.
When delivery batches are pre-negotiated (i.e. when the client selectively tells the server which chunks it would like to receive), we can conclude that the delivery batches are optimsed for _urgency_ rather than for maximising batch utilisation (since the server sends a certain batch that potentially gets reduced into a smaller one by the client before actually being transmitted).

### the protocol defines the notions of:
- **stream** - data source which is composed of a sequence of hashes, referenced by monotonically increasing integers, with unguaranteed contiguity with respect to one particular stream.
- **client** _(downstream peer)_ - the peer which is requesting data and does not posses it (client)
- **server** _(upstream peer)_ - the peer that has the data and sends it to the downstream peer (server)
- **range** - based on the notion of integer indexes we can define a range that designates an interval on the stream.
- **batch** - a set of chunks constituting an interval in a range are called a batch, with a length not exceeding a ceiling value negotiated when establishing streams
- **batch delivery** - end of batch delivery should be indicated by an explicit message from the server
both offered and wanted go together - note this
- **roundtrip** - a configurable extra message exchange (negotiated on initial message exchange) meant to mitigate and avoid requesting the same data from different peers. when the roundtrip is not used the stream is assumed to be continuous and the order of delivery should be guaranteed, or the stream should be particularly ordered. A roundtrip consists of:
    - **offered hashes** - from the server to the client
    - **wanted hashes** - at the discretion of the client in response to offered hashes

### responsibilities:
- client is able to request a range but doesnt know how many results the interval will return from the server
- client does not know if interval is continuous or has gaps
- range is defined by client and should be strictly respected and followed by server
- all intervals specified in protocol messages are closed (inclusive)
- when roundtrip is configured - chunk deliveries can be handled concurrently (therefore their order is not guaranteed), but a server end-of-batch with topmost session index must be sent to signal the end of a batch
- when roundtrip is not configured - chunks are expected to be sent in order, one after the other
- when a client requests an unbounded range (i.e. FROM=..., TO=nil):
    - if there's no chunks available - server waits until something becomes available then send it to the client
    - server's responsibility to give as much as possible, as fast as possible, with a limit of batch size
    - one range query should result in ONE rountrip + batch delivery
- when a client requests a bounded range, server should respond to the client range requests with either offered hashes (if roundtrip is required) or chunks (if not) or an end-of-batch message if there are no more to offer. If none of these responses arrive within a timeout interval, client must drop the upstream peer.
- the server should always respond to the client

 
#### stream termination condition:
 - timeout, connection died, we get an error and remove the client, server also gets an error from p2p layer and removes all servers/clients and drops the peer

### considerations:
- server must make sure that chunk got to client in order to account in SWAP (synchronous). if the send does not result in an error - the send should be accounted
- there is always a max batch size so that clients cannot grieve servers with very large ranges

### syncing contracts:
 - stream indexes always > 0
 - syncing is an implementation of the stream protocol
 - client is expected to manage all intervals, and therefore:
 - server is designed to be stateless, except for the case of managing a offered/wanted roundtrip and the knowledge of a boundedness of a stream (e.g. the server knows that syncing streams are always unbounded from the localstore perspective - data can always enter the system, however this is not the case for live video stream for example)
 - the server does not terminate streams - it is at the discretion of the downstream peer
 - the server does not initiate any messages unless instructed to
 - the server does not instruct client on which bins to subscribe to it

Wire Protocol Specifications
=======

### The wire protocol defines the following messages:

| Msg Name | From->To | Params   | Example |
| -------- | -------- | -------- | ------- |
| StreamInfoReq   | Client->Server  | Streams`[]ID` | `SYNC\|6, SYNC\|5` |
| StreamInfoRes   | Server->Client  | Streams`[]StreamDescriptor` <br>Stream`ID`<br>Cursor`uint64`<br>Bounded`bool` | `SYNC\|6;CUR=1632;bounded, SYNC\|7;CUR=18433;bounded` |
| GetRange | Client->Server| Ruid`uint`<br>Stream `string`<br>From`uint`<br>To`*uint`(nullable)<br>Roundtrip`bool` | `Ruid: 21321, Stream: SYNC\|6, From: 1, To: 100`(bounded), Roundtrip: true<br>`Stream: SYNC\|7, From: 109, Roundtrip: true`(unbounded) | 
| OfferedHashes | Server->Client| Ruid`uint`<br>Hashes `[]byte` | `Ruid: 21321, Hashes: [cbcbbaddda, bcbbbdbbdc, ....]` |
| WantedHashes | Client->Server | Ruid`uint`<br>Bitvector`[]byte` | `Ruid: 21321, Bitvector: [0100100100] ` |
| ChunkDelivery | Server->Client | Ruid`uint`<br>[]Chunk `[]byte` | `Ruid: 21321, Chunk: [001000101]` |
| BatchDone | Server->Client| Ruid `uint`<br>Last `uint` | `Ruid: 21321, Last: 113331` |
| StreamState | Client<->Server | Stream`string`<br>Code`uint16`<br>Message`string`| `Stream: SYNC\|6, Code:1, Message:"Stream became bounded"`<br>`Stream: SYNC\|5, Code:2, Message: "No such stream"` |

Notes:
* communicating the last bin index when roundtrip is configured - can be done on top of OfferedHashes message (alongside the hashes), or to reuse the ACK from the no-roundtrip config
* two notions of bounded - on the stream level and on the localstore
* if TO is not specified - we assume unbounded stream, and we just send whatever, until at most, we fill up an entire batch.

### Message and interface definitions:


```go
// StreamProvider interface provides a lightweight abstraction that allows an easily-pluggable
// stream provider as part of the Stream! protocol specification.
type StreamProvider interface {
  NeedData(ctx context.Context, key []byte) (need bool, wait func(context.Context) error)
	Get(ctx context.Context, addr chunk.Address) ([]byte, error)
	Put(ctx context.Context, addr chunk.Address, data []byte) (exists bool, err error)
	Subscribe(ctx context.Context, key interface{}, from, to uint64) (<-chan chunk.Descriptor, func())
	Cursor(interface{}) (uint64, error)
	RunUpdateStreams(p *Peer)
	StreamName() string
	ParseKey(string) (interface{}, error)
	EncodeKey(interface{}) (string, error)
	StreamBehavior() StreamInitBehavior
	Boundedness() bool
}
```

```go
type StreamInitBehavior int
```

```go
// StreamInfoReq is a request to get information about particular streams
type StreamInfoReq struct {
	Streams []ID
}
```

```go
// StreamInfoRes is a response to StreamInfoReq with the corresponding stream descriptors
type StreamInfoRes struct {
	Streams []StreamDescriptor
}
```

```go
// StreamDescriptor describes an arbitrary stream
type StreamDescriptor struct {
	Stream  ID
	Cursor  uint64
	Bounded bool
}
```

```go
// GetRange is a message sent from the downstream peer to the upstream peer asking for chunks
// within a particular interval for a certain stream
type GetRange struct {
	Ruid      uint
	Stream    ID
	From      uint64
	To        uint64 `rlp:nil`
	BatchSize uint
	Roundtrip bool
}
```

```go
// OfferedHashes is a message sent from the upstream peer to the downstream peer allowing the latter
// to selectively ask for chunks within a particular requested interval
type OfferedHashes struct {
	Ruid      uint
	LastIndex uint
	Hashes    []byte
}
```

```go
// WantedHashes is a message sent from the downstream peer to the upstream peer in response
// to OfferedHashes in order to selectively ask for a particular chunks within an interval
type WantedHashes struct {
	Ruid      uint
	BitVector []byte
}
```

```go
// ChunkDelivery delivers a frame of chunks in response to a WantedHashes message
type ChunkDelivery struct {
	Ruid      uint
	LastIndex uint
	Chunks    []DeliveredChunk
}
```

```go
// DeliveredChunk encapsulates a particular chunk's underlying data within a ChunkDelivery message
type DeliveredChunk struct {
	Addr storage.Address
	Data []byte
}
```

```go
// StreamState is a message exchanged between two nodes to notify of changes or errors in a stream's state
type StreamState struct {
	Stream  ID
	Code    uint16
	Message string
}
```

```go
// Stream defines a unique stream identifier in a textual representation
type ID struct {
	// Name is used for the Stream provider identification
	Name string
	// Key is the name of specific data stream within the stream provider. The semantics of this value
	// is at the discretion of the stream provider implementation
	Key string
}
```

Message exchange examples:
======

Initial handshake - client queries server for stream states<br>
![handshake](https://raw.githubusercontent.com/ethersphere/swarm/master/docs/diagrams/stream-handshake.png)
<br>
GetRange (bounded) - client requests a bounded range within a stream<br>
![bounded-range](https://raw.githubusercontent.com/ethersphere/swarm/master/docs/diagrams/stream-bounded.png)
<br>
GetRange (unbounded) - client requests an unbounded range (specifies only `From` parameter)<br>
![unbounded-range](https://raw.githubusercontent.com/ethersphere/swarm/master/docs/diagrams/stream-unbounded.png)
<br>
GetRange (no roundtrip) - client requests an unbounded or bounded range with no roundtrip configured<br>
![unbounded-range](https://raw.githubusercontent.com/ethersphere/swarm/master/docs/diagrams/stream-no-roundtrip.png)

