// +build none

package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/whisper"
	"github.com/obscuren/secp256k1-go"
)

func main() {
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.InfoLevel))

	pub, sec := secp256k1.GenerateKeyPair()

	whisper := whisper.New(pub, sec)

	srv := p2p.Server{
		MaxPeers:   10,
		Identity:   p2p.NewSimpleClientIdentity("whisper-go", "1.0", "", string(pub)),
		ListenAddr: ":30303",
		NAT:        p2p.UPNP(),

		Protocols: []p2p.Protocol{whisper.Protocol()},
	}
	if err := srv.Start(); err != nil {
		fmt.Println("could not start server:", err)
		os.Exit(1)
	}

	// add seed peers
	seed, err := net.ResolveTCPAddr("tcp", "poc-7.ethdev.com:30300")
	if err != nil {
		fmt.Println("couldn't resolve:", err)
		os.Exit(1)
	}
	srv.SuggestPeer(seed.IP, seed.Port, nil)

	select {}
}
