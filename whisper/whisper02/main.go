// Copyright 2014 The go-ethereum Authors
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

// +build none

// Contains a simple whisper peer setup and self messaging to allow playing
// around with the protocol and API without a fancy client implementation.

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/whisper"
)

func main() {
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.InfoLevel))

	// Generate the peer identity
	key, err := crypto.GenerateKey()
	if err != nil {
		fmt.Printf("Failed to generate peer key: %v.\n", err)
		os.Exit(-1)
	}
	name := common.MakeName("whisper-go", "1.0")
	shh := whisper.New()

	// Create an Ethereum peer to communicate through
	server := p2p.Server{
		PrivateKey: key,
		MaxPeers:   10,
		Name:       name,
		Protocols:  []p2p.Protocol{shh.Protocol()},
		ListenAddr: ":30300",
		NAT:        nat.Any(),
	}
	fmt.Println("Starting Ethereum peer...")
	if err := server.Start(); err != nil {
		fmt.Printf("Failed to start Ethereum peer: %v.\n", err)
		os.Exit(1)
	}

	// Send a message to self to check that something works
	payload := fmt.Sprintf("Hello world, this is %v. In case you're wondering, the time is %v", name, time.Now())
	if err := selfSend(shh, []byte(payload)); err != nil {
		fmt.Printf("Failed to self message: %v.\n", err)
		os.Exit(-1)
	}
}

// SendSelf wraps a payload into a Whisper envelope and forwards it to itself.
func selfSend(shh *whisper.Whisper, payload []byte) error {
	ok := make(chan struct{})

	// Start watching for self messages, output any arrivals
	id := shh.NewIdentity()
	shh.Watch(whisper.Filter{
		To: &id.PublicKey,
		Fn: func(msg *whisper.Message) {
			fmt.Printf("Message received: %s, signed with 0x%x.\n", string(msg.Payload), msg.Signature)
			close(ok)
		},
	})
	// Wrap the payload and encrypt it
	msg := whisper.NewMessage(payload)
	envelope, err := msg.Wrap(whisper.DefaultPoW, whisper.Options{
		From: id,
		To:   &id.PublicKey,
		TTL:  whisper.DefaultTTL,
	})
	if err != nil {
		return fmt.Errorf("failed to seal message: %v", err)
	}
	// Dump the message into the system and wait for it to pop back out
	if err := shh.Send(envelope); err != nil {
		return fmt.Errorf("failed to send self-message: %v", err)
	}
	select {
	case <-ok:
	case <-time.After(time.Second):
		return fmt.Errorf("failed to receive message in time")
	}
	return nil
}
