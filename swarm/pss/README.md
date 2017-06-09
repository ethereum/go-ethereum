# Postal Service over Swarm

pss provides devp2p functionality for swarm nodes without the need for a direct tcp connection between them.

It uses swarm kademlia routing to send and receive messages. Routing is deterministic and will seek the shortest route available on the network.

Messages are encapsulated in a devp2p message structure `PssMsg`. These capsules are forwarded from node to node using ordinary tcp devp2p until it reaches it's destination. The destination address is hinted in `PssMsg.To`

The content of a PssMsg can be anything at all, down to a simple, non-descript byte-slices. But convenience methods are made available to implement devp2p protocol functionality on top of it.

In its final implementation, pss is intended to become "shh over bzz,"  that is; "whisper over swarm." Specifically, this means that the emphemeral encryption envelopes of whisper will be used to obfuscate the correspondance. Ideally, the unencrypted content of the PssMsg will only contain a part of the address of the recipient, where the final recipient is the one who matches this partial address *and* successfully can encrypt the message.

For the current state and roadmap of pss development please see https://github.com/ethersphere/swarm/wiki/swarm-dev-progress.

Please report issues on https://github.com/ethersphere/go-ethereum

Feel free to ask questions in https://gitter.im/ethersphere/pss

## TL;DR IMPLEMENTATION

Most developers will most probably want to use the protocol-wrapping convenience client in swarm/pss/client. Documentation and a minimal code example for the latter is found in the package documentation. The pss API can of course also be used directly. The client implementation provides a clear illustration of its intended usage.

pss implements the node.Service interface. This means that the API methods will be auto-magically exposed to any RPC layer the node activates. In particular, pss provides subscription to incoming messages using the go-ethereum rpc websocket layer. 	

The important API methods are:
- Receive() - start a subscription to receive new incoming messages matching specific "topics"
- Send() - send content over pss to a specified recipient


## LOWLEVEL IMPLEMENTATION

code speaks louder than words:
  
     import (
     	"io/ioutil"
     	"os"
     	"github.com/ethereum/go-ethereum/p2p"
     	"github.com/ethereum/go-ethereum/log"
     	"github.com/ethereum/go-ethereum/swarm/pss"
     	"github.com/ethereum/go-ethereum/swarm/network"
     	"github.com/ethereum/go-ethereum/swarm/storage"
     )
     
     var (
     	righttopic = pss.NewTopic("foo", 4)
     	wrongtopic = pss.NewTopic("bar", 2)
     )
    
     // if you want to see what's going on 
     func init() {
     	hs := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
     	hf := log.LvlFilterHandler(log.LvlTrace, hs)
     	h := log.CallerFileHandler(hf)
     	log.Root().SetHandler(h)
     }
     
     
     // Pss.Handler type
     func handler(msg []byte, p *p2p.Peer, from []byte) error {
     	log.Debug("received", "msg", msg, "from", from, "forwarder", p.ID())
     	return nil
     }
     
     func implementation() {
     
		// bogus addresses for illustration purposes
		meaddr := network.RandomAddr()
		toaddr := network.RandomAddr()
		fwdaddr := network.RandomAddr()
	   
		// new kademlia for routing
		kp := network.NewKadParams()
		to := network.NewKademlia(meaddr.Over(), kp)
	   
		// new (local) storage for cache
		cachedir, err := ioutil.TempDir("", "pss-cache")
		if err != nil {
			panic("overlay")
		}
		dpa, err := storage.NewLocalDPA(cachedir)
		if err != nil {
			panic("storage")
		}
	   
		// setup pss
		psp := pss.NewPssParams(false)
		ps := pss.NewPss(to, dpa, psp)
	   
		// does nothing but please include it
		ps.Start(nil)
	   
		dereg := ps.Register(&righttopic, handler)
	   
		// in its simplest form a message is just a byteslice
		payload := []byte("foobar")
	   
		// send a raw message
		err = ps.SendRaw(toaddr.Over(), righttopic, payload)
		log.Error("Fails. Not connect, so nothing in kademlia. But it illustrates the point.", "err", err)
	   
		// forward a full message
		envfwd := pss.NewEnvelope(fwdaddr.Over(), righttopic, payload)
		msgfwd := &pss.PssMsg{
			To: toaddr.Over(),
			Payload: envfwd,
		}
		err = ps.Forward(msgfwd)
		log.Error("Also fails, same reason. I wish, I wish, I wish there was somebody out there.", "err", err)
	   
		// process an incoming message
		// (this is the first step after the devp2p PssMsg message handler)
		envme := pss.NewEnvelope(toaddr.Over(), righttopic, payload)
		msgme := &pss.PssMsg{
			To: meaddr.Over(),
			Payload: envme,
		}
		err = ps.Process(msgme)
		if err == nil {
			log.Info("this works :)")
		}
	   
		// if we don't have a registered topic it fails
		dereg() // remove the previously registered topic-handler link
		ps.Process(msgme)
		log.Error("It fails as we expected", "err", err)
	   
		// does nothing but please include it
		ps.Stop()
    }

## MESSAGE STRUCTURE

NOTE! This part is subject to change. In particular the envelope structure will be re-implemented using whisper.

A pss message has the following layers:

- PssMsg
   Contains (eventually only part of) recipient address, and (eventually) encrypted Envelope. 

- Envelope
   Currently rlp-encoded. Contains the Payload, along with sender address, topic and expiry information.

- Payload
   Byte-slice of arbitrary data

- ProtocolMsg
   An optional convenience structure for implementation of devp2p protocols. Contains Code, Size and Payload analogous to the p2p.Msg structure, where the payload is a rlp-encoded byteslice. For transport, this struct is serialized and used as the "payload" above.

## TOPICS AND PROTOCOLS

Pure pss is protocol agnostic. Instead it uses the notion of Topic. This is NOT the "subject" of a message. Instead this type is used to internally register handlers for messages matching respective Topics.

Topic in this context virtually mean anything; protocols, chatrooms, or social media groups.

When implementing devp2p protocols, topics are direct mappings to protocols name and version. The pss package provides the PssProtocol convenience structure, and a generic Handler that can be passed to Pss.Register. This makes it possible to use the same message handler code  for pss that are used for direct connected peers.

## CONNECTIONS 

A "connection" in pss is a purely virtual construct. There is no mechanisms in place to ensure that the remote peer actually is there. In fact, "adding" a peer involves merely the node's opinion that the peer is there. It may issue messages to that remote peer to a directly connected peer, which in turn passes it on. But if it is not present on the network - or if there is no route to it - the message will never reach its destination through mere forwarding.

When implementing the devp2p protocol stack, the "adding" of a remote peer is a prerequisite for the side actually initiating the protocol communication. Adding a peer in effect "runs" the protocol on that peer, and adds an internal mapping between a topic and that peer. It also enables sending and receiving messages using the main io-construct in devp2p - the p2p.MsgReadWriter.

Under the hood, pss implements its own MsgReadWriter, which bridges MsgReadWriter.WriteMsg with Pss.SendRaw, and deftly adds an InjectMsg method which pipes incoming messages to appear on the MsgReadWriter.ReadMsg channel.

An incoming connection is nothing more than an actual PssMsg appearing with a certain Topic. If a Handler har been registered to that Topic, the message will be passed to it. This constitutes a "new" connection if:

- The pss node never called AddPeer with this combination of remote peer address and topic, and

- The pss node never received a PssMsg from this remote peer with this specific Topic before.

If it is a "new" connection, the protocol will be "run" on the remote peer, in the same manner as if it was pre-emptively added.

## ROUTING AND CACHING

(please refer to swarm kademlia routing for an explanation of the routing algorithm used for pss)

pss implements a simple caching mechanism, using the swarm DPA for storage of the messages and generation of the digest keys used in the cache table. The caching is intended to alleviate the following:

- save messages so that they can be delivered later if the recipient was not online at the time of sending.

- drop an identical message to the same recipient if received within a given time interval

- prevent backwards routing of messages

the latter may occur if only one entry is in the receiving node's kademlia. In this case the forwarder will be provided as the "nearest node" to the final recipient. The cache keeps the address of who the message was forwarded from, and if the cache lookup matches, the message will be dropped.
