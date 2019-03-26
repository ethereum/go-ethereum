# Postal Services over Swarm

`pss` enables message relay over swarm. This means nodes can send messages to each other without being directly connected with each other, while taking advantage of the efficient routing algorithms that swarm uses for transporting and storing data.

### CONTENTS

* Status of this document
* Core concepts
* Caveat
* Examples
* API
  * Retrieve node information
  * Receive messages
  * Send messages using public key encryption
  * Send messages using symmetric encryption
  * Querying peer keys
  * Handshakes

### STATUS OF THIS DOCUMENT

`pss` is under active development, and the first implementation is yet to be merged to the Ethereum main branch. Expect things to change.

Details on swarm routing and encryption schemes out of scope of this document.

Please refer to [ARCHITECTURE.md](ARCHITECTURE.md) for in-depth topics concerning `pss`.

## CORE CONCEPTS

Three things are required to send a `pss` message:

1. Encryption key
2. Topic
3. Message payload

Encryption key can be a public key or a 32 byte symmetric key. It must be coupled with a peer address in the node prior to sending.

Topic is the initial 4 bytes of a hash value.

Message payload is an arbitrary byte slice of data.

Upon sending the message it is encrypted and passed on from peer to peer. Any node along the route that can successfully decrypt the message is regarded as a recipient. Recipients continue to pass on the message to their peers, to make traffic analysis attacks more difficult.

The Address that is coupled with the encryption keys are used for routing the message. This does *not* need to be a full addresses; the network will route the message to the best of its ability with the information that is available. If *no* address is given (zero-length byte slice), routing is effectively deactivated, and the message is passed to all peers by all peers.

## CAVEAT

`pss` connectivity resembles UDP. This means there is no delivery guarantee for a message. Furthermore there is no strict definition of what a connection between two nodes communicating via `pss` is. Reception acknowledgements and keepalive-schemes is the responsibility of the application.

Due to the inherent properties of the `swarm` routing algorithm, a node may receive the same message more than once. Message deduplication *cannot be guaranteed* by `pss`, and must be handled in the application layer to ensure predictable results.

## EXAMPLES

The code tutorial [p2p programming in go-ethereum](https://github.com/nolash/go-ethereum-p2p-demo) by [@nolash](https://github.com/nolash) provides step-by-step code examples for usage of `pss` API with `go-ethereum` nodes.

A quite unpolished example using `javascript` is available here: [https://github.com/nolash/pss-js/tree/withcrypt](https://github.com/nolash/pss-js/tree/withcrypt)

## API

The `pss` API is available through IPC and Websockets. There is currently no `web3.js` implementation, as this does not support message subscription.

For `golang` clients, please use the `rpc.Client` provided by the `go-ethereum` repository. The return values may have special types in `golang`. Please refer to `godoc` for details.

### RETRIEVE NODE INFORMATION

#### pss_getPublicKey

Retrieves the public key of the node, in hex format

```
parameters:
none

returns:
1. publickey (hex)
```

#### pss_baseAddr

Retrieves the swarm overlay address of the node, in hex format

```
parameters:
none

returns:
1. swarm overlay address (hex)
```

#### pss_stringToTopic

Creates a deterministic 4 byte topic value from input, returned in hex format

```
parameters:
1. topic string (string)

returns:
1. pss topic (hex)
```

### RECEIVE MESSAGES

#### pss_subscribe

Creates a subscription. Received messages with matching topic will be passed to subscription client.

```
parameters:
1. string("receive")
2. topic (4 bytes in hex)

returns:
1. subscription handle `base64(byte)` `rpc.ClientSubscription`
```

In `golang` as special method is used:

`rpc.Client.Subscribe(context.Context, "pss", chan pss.APIMsg, "receive", pss.Topic)`

Incoming messages are encapsulated in an object (`pss.APIMsg` in `golang`) with the following members:

```
1. Msg (hex) - the message payload
2. Asymmetric (bool) - true if message used public key encryption
3. Key (string) - the encryption key used
```

### SEND MESSAGE USING PUBLIC KEY ENCRYPTION

#### pss_setPeerPublicKey

Register a peer's public key. This is done once for every topic that will be used with the peer. Address can be anything from 0 to 32 bytes inclusive of the peer's swarm overlay address.

```
parameters:
1. public key of peer (hex)
2. topic (4 bytes in hex)
3. address of peer (hex)

returns:
none
```

#### pss_sendAsym

Encrypts the message using the provided public key, and signs it using the node's private key. It then wraps it in an envelope containing the topic, and sends it to the network. 

```
parameters:
1. public key of peer (hex)
2. topic (4 bytes in hex)
3. message (hex)

returns:
none
```

### SEND MESSAGE USING SYMMETRIC ENCRYPTION

#### pss_setSymmetricKey

Register a symmetric key shared with a peer. This is done once for every topic that will be used with the peer. Address can be anything from 0 to 32 bytes inclusive of the peer's swarm overlay address.

If the fourth parameter is false, the key will *not* be added to the list of symmetric keys used for decryption attempts.

```
parameters:
1. symmetric key (hex)
2. topic (4 bytes in hex)
3. address of peer (hex)
4. use for decryption (bool)

returns:
1. symmetric key id (string)
```

#### pss_sendSym

Encrypts the message using the provided symmetric key, wraps it in an envelope containing the topic, and sends it to the network.

```
parameters:
1. symmetric key id (string)
2. topic (4 bytes in hex)
3. message (hex)

returns:
none
```

### QUERY PEER KEYS

#### pss_GetSymmetricAddressHint

Return the swarm overlay address associated with the peer registered with the given symmetric key and topic combination.

```
parameters:
1. topic (4 bytes in hex)
2. symmetric key id (string)

returns:
1. peer address (hex)
```

#### pss_GetAsymmetricAddressHint

Return the swarm overlay address associated with the peer registered with the given symmetric key and topic combination.

```
parameters:
1. topic (4 bytes in hex)
2. public key in hex form (string)

returns:
1. peer address (hex)
```

### HANDSHAKES

Convenience implementation of Diffie-Hellman handshakes using ephemeral symmetric keys. Peers keep separate sets of keys for incoming and outgoing communications.

*This functionality is an optional feature in `pss`. It is compiled in by default, but can be omitted by providing the `nopsshandshake` build tag.*

#### pss_addHandshake

Activate handshake functionality on the specified topic.

```
parameters:
1. topic (4 bytes in hex)

returns:
none
```

#### pss_removeHandshake

Remove handshake functionality on the specified topic.

```
parameters:
1. topic (4 bytes in hex)

returns:
none
```

#### pss_handshake

Instantiate handshake with peer, refreshing symmetric encryption keys.

If parameter 3 is false, the returned array will be empty.

```
parameters:
1. public key of peer in hex format (string)
2. topic (4 bytes in hex)
3. block calls until keys are received (bool)
4. flush existing incoming keys (bool)

returns:
1. list of symmetric keys (string[])
```

#### pss_getHandshakeKeys

Get valid symmetric encryption keys for a specified peer and topic.

parameters:
1. public key of peer in hex format (string)
2. topic (4 bytes in hex)
3. include keys for incoming messages (bool)
4. include keys for outgoing messages (bool)

returns:
1. list of symmetric keys (string[])

#### pss_getHandshakeKeyCapacity

Get amount of remaining messages the specified key is valid for.

```
parameters:
1. symmetric key id (string)

returns:
1. number of messages (uint16)
```

#### pss_getHandshakePublicKey

Get the peer's public key associated with the specified symmetric key.

```
parameters:
1. symmetric key id (string)

returns:
1. Associated public key in hex format (string)
```

#### pss_releaseHandshakeKey

Invalidate the specified key.

Normally, the key will be kept for a grace period to allow for decryption of delayed messages. If instant removal is set, this grace period is omitted, and the key removed instantaneously.

```
parameters:
1. public key of peer in hex format (string)
2. topic (4 bytes in hex)
3. symmetric key id to release (string)
4. remove keys instantly (bool)

returns:
1. whether key was successfully removed (bool)
```
