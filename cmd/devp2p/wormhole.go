// Copyright 2022 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/urfave/cli/v2"
)

func discv5WormholeSend(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	disc := startV5(ctx)
	defer disc.Close()
	fmt.Println(disc.Ping(n))
	resp, err := disc.TalkRequest(n, "wrm", []byte("rand"))
	log.Info("Talkrequest", "resp", fmt.Sprintf("%v (%x)", string(resp), resp), "err", err)

	return nil
}

func discv5WormholeReceive(ctx *cli.Context) error {
	var unhandled chan discover.ReadPacket
	disc := startV5WithUnhandled(ctx, unhandled)
	defer disc.Close()

	fmt.Println(disc.Self())

	disc.RegisterTalkHandler("wrm", handleWormholeTalkrequest)
	handleUnhandledLoop(unhandled)
	return nil
}

// TalkRequestHandler callback processes a talk request and optionally returns a reply
//type TalkRequestHandler func(enode.ID, *net.UDPAddr, []byte) []byte

func handleWormholeTalkrequest(id enode.ID, addr *net.UDPAddr, data []byte) []byte {
	log.Info("Handling talk request", "from", addr, "id", id, "data", fmt.Sprintf("%x", data))
	return []byte("oll korrekt!")
}

func handleUnhandledLoop(unhandled chan discover.ReadPacket) {
	for {
		select {
		case packet := <-unhandled:
			log.Info("Unhandled packet handled", "from", packet.Addr, "data", fmt.Sprintf("%v %#x", string(packet.Data), packet.Data))
		}
	}
}
