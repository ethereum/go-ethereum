The peer to peer package ([go-ethereum/p2p](https://github.com/ethereum/go-ethereum/tree/develop/p2p)) allows you to rapidly and easily add peer to peer networking to any type of application. The p2p package is set up in a modular structure and extending the p2p with your own additional sub protocols is easy and straight forward.

Starting the p2p service only requires you setup a `p2p.Server{}` with a few settings:

```go
import "github.com/ethereum/go-ethereum/crypto"
import "github.com/ethereum/go-ethereum/p2p"

nodekey, _ := crypto.GenerateKey()
srv := p2p.Server{
	MaxPeers:   10,
	PrivateKey: nodekey,
	Name:       "my node name",
	ListenAddr: ":30300",
	Protocols:  []p2p.Protocol{},
}
srv.Start()
```

If we wanted to extend the capabilities of our p2p server we'd need to pass it an additional sub protocol in the `Protocol: []p2p.Protocol{}` array. 

An additional sub protocol that has the ability to respond to the message "foo" with "bar" requires you to setup an `p2p.Protocol{}`:

```go
func MyProtocol() p2p.Protocol {
	return p2p.Protocol{ // 1.
		Name:    "MyProtocol",                                                    // 2.
		Version: 1,                                                               // 3.
		Length:  1,                                                               // 4.
		Run:     func(peer *p2p.Peer, ws p2p.MsgReadWriter) error { return nil }, // 5.
	}
}
```

1. A sub-protocol object in the p2p package is called `Protocol{}`. Each time a peer connects with the capability of handling this type of protocol will use this;
2. The name of your protocol to identify the protocol on the network;
3. The version of the protocol.
4. The amount of messages this protocol relies on. Because the p2p is extendible and thus has the ability to send an arbitrary amount of messages (with a type, which we'll see later) the p2p handler needs to know how much space it needs to reserve for your protocol, this to ensure consensus can be reached between the peers doing a negotiation over the message IDs. Our protocol supports only one; `message` (as you'll see later).
5. The main handler of your protocol. We've left this intentionally blank for now. The `peer` variable is the peer connected to you and provides you with some basic information regarding the peer. The `ws` variable which is a reader and a writer allows you to communicate with the peer. If a message is being send to us by that peer the `MsgReadWriter` will handle it and vice versa.

Lets fill in the blanks and create a somewhat useful peer by allowing it to communicate with another peer:

```go
const messageId = 0   // 1.
type Message string   // 2.

func msgHandler(peer *p2p.Peer, ws p2p.MsgReadWriter) error {
    for {
        msg, err := ws.ReadMsg()   // 3.
        if err != nil {            // 4.
            return err // if reading fails return err which will disconnect the peer.
        }

        var myMessage [1]Message
        err = msg.Decode(&myMessage) // 5.
        if err != nil {
            // handle decode error
            continue
        }
        
        switch myMessage[0] {
        case "foo":
            err := p2p.SendItems(ws, messageId, "bar")  // 6.
            if err != nil {
                return err // return (and disconnect) error if writing fails.
            }
         default:
             fmt.Println("recv:", myMessage)
         }
    }

    return nil
}
```

1. The one and only message we know about;
2. A typed string we decode in to;
3. `ReadMsg` waits on the line until it receives a message, an error or EOF.
4. In case of an error during reading it's best to return that error and let the p2p server handle it. This usually results in a disconnect from the peer.
5. `msg` contains two fields and a decoding method:
    * `Code` contains the message id, `Code == messageId` (i.e., 0)
    * `Payload` the contents of the message.
    * `Decode(<ptr>)` is a helper method for: take `msg.Payload` and decodes the rest of the message in to the given interface. If it fails it will return an error.
6. If the message we decoded was `foo` respond with a `NewMessage` using the `messageId` message identifier and respond with the message `bar`. The `bar` message would be handled in the `default` case in the same switch.

Now if we'd tie this all up we'd have a working p2p server with a message passing sub protocol.

```go
package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
)

const messageId = 0

type Message string

func MyProtocol() p2p.Protocol {
	return p2p.Protocol{
		Name:    "MyProtocol",
		Version: 1,
		Length:  1,
		Run:     msgHandler,
	}
}

func main() {
	nodekey, _ := crypto.GenerateKey()
	srv := p2p.Server{
		MaxPeers:   10,
		PrivateKey: nodekey,
		Name:       "my node name",
		ListenAddr: ":30300",
		Protocols:  []p2p.Protocol{MyProtocol()},
	}

	if err := srv.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	select {}
}

func msgHandler(peer *p2p.Peer, ws p2p.MsgReadWriter) error {
	for {
		msg, err := ws.ReadMsg()
		if err != nil {
			return err
		}

		var myMessage Message
		err = msg.Decode(&myMessage)
		if err != nil {
			// handle decode error
			continue
		}

		switch myMessage {
		case "foo":
			err := p2p.SendItems(ws, messageId, "bar"))
			if err != nil {
				return err
			}
		default:
			fmt.Println("recv:", myMessage)
		}
	}

	return nil
}
```
   