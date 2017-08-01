p2p/protocols: devp2p subprotocol abstraction

The protocols subpackage is an extension to p2p. It offers a simple and user friendly simple way
to define devp2p subprotocols by abstracting away code that implementations would typically share.

The package provides a protocol peer object of type protocols.Peer initialised from

* a p2p.Peer, a p2p.MsgReadWriter (the arguments passed to p2p.Protocol#Run),
* a protocols.CodeMap, this encodes the msg code and msg type associations
* messenger interface (with methods SendMsg and ReadMsg) that abstracts out sending and receiving a msg
* disconnect function

Allowing the p2p.Protocol#Run function to construct this peer allows passing it to arbitrary
service instances sitting on peer connections. These service instances can encapsulate vertical slices
of business logic without duplicating code related to protocol communication.

Features

* registering multiple handler callbacks for incoming messages
* automate RLP decoding/encoding based on reflection
* provide the forever loop to read incoming messages
* standardise error handling related to communication
* with disconnection and messaging abstracted out allows protocols to be used
  in network simulations with or without serialisation, transport and p2p server
* TODO: automatic generation of wire protocol specification for peers

see the possibly obsolete #2254 for the peer management/connectivity related aspect)
