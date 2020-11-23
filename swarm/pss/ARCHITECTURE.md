# Postal Service over Swarm

Pss provides devp2p functionality for swarm nodes without the need for a direct tcp connection between them.

Messages are encapsulated in a devp2p message structure `PssMsg`. These capsules are forwarded from node to node using ordinary tcp devp2p until they reach their destination: The node or nodes who can successfully decrypt the message.

| Layer     | Contents        |
|-----------|-----------------|
| PssMsg:   | Address, Expiry |
| Envelope: | Topic           |
| Payload:  | e(data)         |

Routing of messages is done using swarm's own kademlia routing. Optionally routing can be turned off, forcing the message to be sent to all peers, similar to the behavior of the whisper protocol.

Pss is intended for messages of limited size, typically a couple of Kbytes at most. The messages themselves can be anything at all; complex data structures or non-descript byte sequences.

For the current state and roadmap of pss development please see https://github.com/ethersphere/swarm/wiki/swarm-dev-progress.

Please report issues on https://github.com/ethersphere/go-ethereum

Feel free to ask questions in https://gitter.im/ethersphere/pss

## STATUS OF THIS DOCUMENT

`pss` is under active development, and the first implementation is yet to be merged to the Ethereum main branch. Expect things to change.

## CORE INTERFACES

The pss core provides low level control of key handling and message exchange. 

### TOPICS

An encrypted envelope of a pss message always contains a Topic. This is pss' way of determining which message handlers to dispatch messages to. The topic of a message is only visible for the node(s) who can decrypt the message.

This "topic" is not like the subject of an email message, but a hash-like arbitrary 4 byte value. A valid topic can be generated using the `pss_*ToTopic` API methods.

### IDENTITY AND ENCRYPTION

Pss aims to achieve perfect darkness. That means that the minimum requirement for two nodes to communicate using pss is a shared secret. This secret can be an arbitrary byte slice, or a ECDSA keypair. The end recipient of a message is defined as the node that can successfully decrypt that message using stored keys.

A node's public key is derived from the private key passed to the `pss` constructor. Pss (currently) has no PKI.

Peer keys can manually be added to the pss node through its API calls `pss_setPeerPublicKey` and `pss_setSymmetricKey`. Keys are always coupled with a topic, and the keys will only be valid for these topics.

### CONNECTIONS

A "connection" in pss is a purely virtual construct. There is no mechanisms in place to ensure that the remote peer actually is there. In fact, "adding" a peer involves merely the node's opinion that the peer is there. It may issue messages to that remote peer to a directly connected peer, which in turn passes it on. But if it is not present on the network - or if there is no route to it - the message will never reach its destination through mere forwarding.

Since pss itself never requires a confirmation from a peer of whether a message is received or not, one could argue that pss shows `UDP`-like behavior.

It is also important to note that if the wrong (partial) address is set for a particular key/topic combination, the message may never reach that peer. The further left in the address byte slice the error lies, the less likely it is that delivery will occur. 


### EXCHANGE

Message exchange in `pss` *requires* end-to-end encryption. 

The API methods `pss_sendSym` and `pss_sendAsym` sends an arbitrary byte slice with a specific topic to a pss peer using the respective encryption scheme. The key passed to the send method must be associated with a topic in the pss key store prior to sending, or the send method will fail.

Return values from the send methods do *not* indicate whether the message was successfully delivered to the pss peer. It *only* indicates whether or not the message could be passed on to the network. If the message could not be forwarded to any peers, the method will fail.

Keep in mind that symmetric encryption is less resource-intensive than asymmetric encryption. The former should be used for nodes with high message volumes.

## EXTENSIONS

### HANDSHAKE

Pss offers an optional Diffie-Hellman handshake mechanism. Handshake functionality is activated per topic, and can be deactivated per topic even while the node is running.

Handshakes are activated in the code implementation of the node by running `SetHandshakeController()` on the pss node instance BEFORE starting the node service. The methods exposed by the HandshakeController's API gives the possibility to initiate, remove and check the state of handshakes and associated keys.

See the `HandshakeAPI` section in `godoc` for details.

### DEVP2P PROTOCOLS

The `Protocol` convenience structure is provided to mimic devp2p-type protocols over pss. In theory this makes it possible to reuse protocol code written for devp2p with a minimum of effort.

#### OUTGOING CONNECTIONS

In order to message a peer using this layer, a `Protocol` object must first be instantiated. When this is done, peers can be added using the protocol's `AddPeer()` method. The peer's key/topic combination must be in the pss key store before the peer can be aded.

Adding a peer in effect "runs" the protocol on that peer, and adds an internal mapping between a topic and that peer, and enables sending and receiving messages using the usual io-construct of devp2p. It does not actually *transmit* anything to the peer, it merely represents the node's opinion that a connection with the peer exists. (See CONNECTION above).

#### INCOMING CONNECTIONS

An incoming connection is nothing more than an actual PssMsg appearing with a certain Topic. If a Handler has been registered to that Topic, the message will be passed to it. This constitutes a "new" connection if:

- The pss node never called AddPeer with this combination of remote peer address and topic, and

- The pss node never received a PssMsg from this remote peer with this specific Topic before.

If it is a "new" connection, the protocol will be "run" on the remote peer, as if the peer was added via the API. 

As with the `AddPeer()` method, the key/topic of the originating peer must exist in the pss key store.

#### TOPICS IN DEVP2P

The `ProtocolTopic()` method should be used to determine the correct topic to use for a pss `Protocol` instance.

## EXAMPLES

Coming. Please refer to the tests for now.

## PSS INTERNALS

Pss implements the node.Service interface. It depends on a working kademlia overlay for routing.

### DECRYPTION

When processing an incoming message, `pss` detects whether it is encrypted symmetrically or asymmetrically.

When decrypting symmetrically, `pss` iterates through all stored keys, and attempts to decrypt with each key in order.

pss keeps a *cache* of these keys. The cache will only store a certain amount of keys, and the iterator will return keys in the order of most recently used key first. Abandoned keys will be garbage collected.

### ROUTING 

(please refer to swarm kademlia routing for an explanation of the routing algorithm used for pss)

`pss` uses *address hinting* for routing. The address hint is an arbitrary-length MSB byte slice of the peer's swarm overlay address. It can be the whole address, part of the address, or even an empty byte slice. The slice will be matched to the MSB slice of the same length of all devp2p peers in the routing stage.

If an empty byte slice is passed, all devp2p peers will match the address hint, and the message will be forwarded to everyone. This is equivalent to `whisper` routing, and makes it difficult to perform traffic analysis based on who messages are forwarded to.

A node will also forward to everyone if the address hint provided is in its proximity bin, both to provide saturation to increase chances of delivery, and also for recipient obfuscation to thwart traffic analysis attacks. The recipient node(s) will always forward to all its peers.

### CACHING

pss implements a simple caching mechanism for messages, using the swarm FileStore for storage of the messages and generation of the digest keys used in the cache table. The caching is intended to alleviate the following:

- save messages so that they can be delivered later if the recipient was not online at the time of sending.

- drop an identical message to the same recipient if received within a given time interval

- prevent backwards routing of messages

the latter may occur if only one entry is in the receiving node's kademlia, or if the proximity of the current node recipient hinted by the address is so close that the message will be forwarded to everyone. In these cases the forwarder will be provided as the "nearest node" to the final recipient. The cache keeps the address of who the message was forwarded from, and if the cache lookup matches, the message will be dropped.

### DEVP2P PROTOCOLS

When implementing devp2p protocols, topics are derived from protocols' name and version. The Protocol provides a generic Handler that be passed to Pss.Register. This makes it possible to use the same message handler code for pss that is used for directly connected peers in devp2p.

Under the hood, pss implements its own MsgReadWriter, which bridges MsgReadWriter.WriteMsg with Pss.SendRaw, and deftly adds an InjectMsg method which pipes incoming messages to appear on the MsgReadWriter.ReadMsg channel.


