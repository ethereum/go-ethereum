## PSS tests failures explanation

This document aims to explain the changes in https://github.com/ethersphere/go-ethereum/pull/126 and how those changes affect the pss_test.go TestNetwork tests.

### Problem

When running the TestNetwork test, execution sometimes:

* deadlocks
* panics
* failures with wrong result, such as:

```
$ go test -v ./swarm/pss -cpu 4 -run TestNetwork
```

```
--- FAIL: TestNetwork (68.13s)
    --- FAIL: TestNetwork/3/10/4/sim (68.13s)
        pss_test.go:697: 7 of 10 messages received
        pss_test.go:700: 3 messages were not received
FAIL
```

Moreover execution almost always deadlocks with `sim` adapter, and `sock` adapter (when buffer is low), but is mostly stable with `exec` and `tcp` adapters.

### Findings and Fixes

#### 1. Addressing panics

Panics were caused due to concurrent map read/writes and unsynchronised access to shared memory by multiple goroutines. This is visible when running the test with the `-race` flag.

```
go test -race -v ./swarm/pss -cpu 4 -run TestNetwork

  1 ==================
  2 WARNING: DATA RACE
  3 Read at 0x00c424d456a0 by goroutine 1089:
  4   github.com/ethereum/go-ethereum/swarm/pss.(*Pss).forward.func1()
  5       /Users/nonsense/code/src/github.com/ethereum/go-ethereum/swarm/pss/pss.go:654 +0x44f
  6   github.com/ethereum/go-ethereum/swarm/network.(*Kademlia).eachConn.func1()
  7       /Users/nonsense/code/src/github.com/ethereum/go-ethereum/swarm/network/kademlia.go:350 +0xc9
  8   github.com/ethereum/go-ethereum/pot.(*Pot).eachNeighbour.func1()
  9       /Users/nonsense/code/src/github.com/ethereum/go-ethereum/pot/pot.go:599 +0x59
  ...

 28
 29 Previous write at 0x00c424d456a0 by goroutine 829:
 30   github.com/ethereum/go-ethereum/swarm/pss.(*Pss).Run()
 31       /Users/nonsense/code/src/github.com/ethereum/go-ethereum/swarm/pss/pss.go:192 +0x16a
 32   github.com/ethereum/go-ethereum/swarm/pss.(*Pss).Run-fm()
 33       /Users/nonsense/code/src/github.com/ethereum/go-ethereum/swarm/pss/pss.go:185 +0x63
 34   github.com/ethereum/go-ethereum/p2p.(*Peer).startProtocols.func1()
 35       /Users/nonsense/code/src/github.com/ethereum/go-ethereum/p2p/peer.go:347 +0x8b
 ...
```

##### Current solution

Adding a mutex around all shared data.

#### 2. Failures with wrong result

The validation phase of the TestNetwork test is done using an RPC subscription:

```
    ...
	triggerChecks := func(trigger chan enode.ID, id enode.ID, rpcclient *rpc.Client) error {
		msgC := make(chan APIMsg)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		sub, err := rpcclient.Subscribe(ctx, "pss", msgC, "receive", hextopic)
		...
```

By design the RPC uses a subscription buffer with a max length. When this length is reached, the subscription is dropped. The current config value is not suitable for stress tests.

##### Current solution

Increase the max length of the RPC subscription buffer.

```
const (
	// Subscriptions are removed when the subscriber cannot keep up.
	//
	// This can be worked around by supplying a channel with sufficiently sized buffer,
	// but this can be inconvenient and hard to explain in the docs. Another issue with
	// buffered channels is that the buffer is static even though it might not be needed
	// most of the time.
	//
	// The approach taken here is to maintain a per-subscription linked list buffer
	// shrinks on demand. If the buffer reaches the size below, the subscription is
	// dropped.
	maxClientSubscriptionBuffer = 20000
)
```

#### 3. Deadlocks

Deadlocks are triggered when using:
* `sim` adapter - synchronous, unbuffered channel
* `sock` adapter - asynchronous, buffered channel (when using a 1K buffer)

No deadlocks were triggered when using:
* `tcp` adapter - asynchronous, buffered channel
* `exec` adapter - asynchronous, buffered channel

Ultimately the deadlocks happen due to blocking `pp.Send()` call at:

 		 // attempt to send the message
  		err := pp.Send(msg)
  		if err != nil {
  			log.Debug(fmt.Sprintf("%v: failed forwarding: %v", sendMsg, err))
  			return true
  		}

 `p2p` request handling is synchronous (as discussed at https://github.com/ethersphere/go-ethereum/issues/130), `pss` is also synchronous, therefore if two nodes happen to be processing a request, while at the same time waiting for response on `pp.Send(msg)`, deadlock occurs.
 
 `pp.Send(msg)` is only blocking when the underlying adapter is blocking (read `sim` or `sock`) or the buffer of the connection is full.
 
##### Current solution

Make no assumption on the undelying connection, and call `pp.Send` asynchronously in a go-routine.

Alternatively, get rid of the `sim` and `sock` adapters, and use `tcp` adapter for testing.
