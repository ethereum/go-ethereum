Whisper is a pure identity-based messaging system. Whisper provides a low-level (non-application-specific) but easily-accessible API without being based upon or prejudiced by the low-level hardware attributes and characteristics, particularly the notion of singular endpoints.

This tutorial assumes you've read [p2p 101](https://github.com/ethereum/go-ethereum/wiki/Peer-to-Peer). If you haven't read it I suggest you read it. This tutorial will guide you to setting up a full p2p server with whisper capabilities.

Let's quickly cover some of whisper's basic functionality and discuss it in greater detail later.

```go
whisper.Send(myEnvelope)
```

The notion of envelopes and messages in whisper is somewhat blurred. An application shouldn't ever need to know the difference between the two and should only care about the information it's interested in. Therefor whisper comes with a subscribing mechanism which allows you watch/listen for specific whisper messages (e.g., to you, with a specific topic, etc).

```go
whisper.Watch(Filter{
        From: myFriendsPubKey,
        Fn: func(msg *whisper.Message) { /* ... */ },
})
```

## Envelopes & Messages

Whenever you want to send message over the whisper network you need to prove to network you've done some significant work for sealing the message (such is the cost for sending messages) and thus the more work you put in to sealing the message the higher the priority the message will have when propagating it over the network.

Whisper's *P*roof *o*f *W*ork consists of a simple SHA3 algorithm in which we try to find the smallest number within a given time frame. Giving the algorithm more time will result in a smaller number which means the message has a higher priority in the network.

Messages are also sealed with a *T*ime *T*o *L*ive. Whisper peers will automatically flush out messages which have exceeded their time to live (with a maximum up to 2 days).

Additionally messages may also contain a recipient (public key) and a set of topics. Topics will allow us to specify messages their subject (e.g., "shoes", "laptop", "marketplace", "chat"). Topics are automatically hashed and only the first 4 bytes are used during transmission and as such, topics are not 100% reliable, they should be treated as a probabilistic message filter.

Sending a whisper message requires you to:

1. create a new `whisper.Message`
2. `Seal` it (optionally encrypt, sign and supply with topics)
3. `Send` it to your peers

```go
topics := TopicsFromString("my", "message")
msg := whisper.NewMessage([]byte("hello world"))  // 1
envelope := msg.Seal(whisper.Opts{                // 2
        From:   myPrivateKey, // Sign it
        Topics: topics,
})
whisper.Send(envelope)                            // 3
```

Whenever a message needs to be encrypted for a specific recipient supply the `Opts` struct with an additional `To` parameter which accepts the recipients public key (`ecdsa.PublicKey`).

## Watching & Listening

Watching for specific messages on the whisper network can be done using the `Watch` method. You have the option to watch for messages from a specific recipient, with specific topics or messages directly directed to you.

```go
topics := TopicsFromString("my", "message")
whisper.Watch(Filter{
        Topics: topics
        Fn:     func(msg *Message) {
                fmt.Println(msg)
        },
})
```

## Connecting it all together

Now if we tie it all together and supply whisper as a sub-protocol to the DEV's P2P service we have whisper including peer handling and message propagation.

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/whisper"
	"github.com/obscuren/secp256k1-go"
)

func main() {
	pub, _ := secp256k1.GenerateKeyPair()

	whisper := whisper.New()

	srv := p2p.Server{
		MaxPeers:   10,
		Identity:   p2p.NewSimpleClientIdentity("my-whisper-app", "1.0", "", string(pub)),
		ListenAddr: ":8000",
		Protocols: []p2p.Protocol{whisper.Protocol()},
	}
	if err := srv.Start(); err != nil {
		fmt.Println("could not start server:", err)
		os.Exit(1)
	}

	select {}
}
```