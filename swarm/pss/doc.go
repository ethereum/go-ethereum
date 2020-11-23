// Copyright 2018 The go-ethereum Authors
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

// Pss provides devp2p functionality for swarm nodes without the need for a direct tcp connection between them.
//
// Messages are encapsulated in a devp2p message structure `PssMsg`. These capsules are forwarded from node to node using ordinary tcp devp2p until it reaches its destination: The node or nodes who can successfully decrypt the message.
//
// Routing of messages is done using swarm's own kademlia routing. Optionally routing can be turned off, forcing the message to be sent to all peers, similar to the behavior of the whisper protocol.
//
// Pss is intended for messages of limited size, typically a couple of Kbytes at most. The messages themselves can be anything at all; complex data structures or non-descript byte sequences.
//
// Documentation can be found in the README file.
//
// For the current state and roadmap of pss development please see https://github.com/ethersphere/swarm/wiki/swarm-dev-progress.
//
// Please report issues on https://github.com/ethersphere/go-ethereum
//
// Feel free to ask questions in https://gitter.im/ethersphere/pss
//
// TOPICS
//
// An encrypted envelope of a pss messages always contains a Topic. This is pss' way of determining what action to take on the message. The topic is only visible for the node(s) who can decrypt the message.
//
// This "topic" is not like the subject of an email message, but a hash-like arbitrary 4 byte value. A valid topic can be generated using the `pss_*ToTopic` API methods.
//
// IDENTITY IN PSS
//
// Pss aims to achieve perfect darkness. That means that the minimum requirement for two nodes to communicate using pss is a shared secret. This secret can be an arbitrary byte slice, or a ECDSA keypair.
//
// Peer keys can manually be added to the pss node through its API calls `pss_setPeerPublicKey` and `pss_setSymmetricKey`. Keys are always coupled with a topic, and the keys will only be valid for these topics.
//
// CONNECTIONS
//
// A "connection" in pss is a purely virtual construct. There is no mechanisms in place to ensure that the remote peer actually is there. In fact, "adding" a peer involves merely the node's opinion that the peer is there. It may issue messages to that remote peer to a directly connected peer, which in turn passes it on. But if it is not present on the network - or if there is no route to it - the message will never reach its destination through mere forwarding.
//
// When implementing the devp2p protocol stack, the "adding" of a remote peer is a prerequisite for the side actually initiating the protocol communication. Adding a peer in effect "runs" the protocol on that peer, and adds an internal mapping between a topic and that peer. It also enables sending and receiving messages using the main io-construct in devp2p - the p2p.MsgReadWriter.
//
// Under the hood, pss implements its own MsgReadWriter, which bridges MsgReadWriter.WriteMsg with Pss.SendRaw, and deftly adds an InjectMsg method which pipes incoming messages to appear on the MsgReadWriter.ReadMsg channel.
//
// An incoming connection is nothing more than an actual PssMsg appearing with a certain Topic. If a Handler har been registered to that Topic, the message will be passed to it. This constitutes a "new" connection if:
//
// - The pss node never called AddPeer with this combination of remote peer address and topic, and
//
// - The pss node never received a PssMsg from this remote peer with this specific Topic before.
//
// If it is a "new" connection, the protocol will be "run" on the remote peer, in the same manner as if it was pre-emptively added.
//
package pss
