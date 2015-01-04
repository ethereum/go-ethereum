package main

import (
	"crypto/elliptic"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
)

func main() {
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.InfoLevel))
	key, _ := crypto.GenerateKey()
	marshaled := elliptic.Marshal(crypto.S256(), key.PublicKey.X, key.PublicKey.Y)

	srv := p2p.Server{
		MaxPeers:   100,
		Identity:   p2p.NewSimpleClientIdentity("Ethereum(G)", "0.1", "Peer Server Two", string(marshaled)),
		ListenAddr: ":30301",
		NAT:        p2p.UPNP(),
	}
	if err := srv.Start(); err != nil {
		fmt.Println("could not start server:", err)
		os.Exit(1)
	}

	// add seed peers
	seed, err := net.ResolveTCPAddr("tcp", "poc-8.ethdev.com:30303")
	if err != nil {
		fmt.Println("couldn't resolve:", err)
	} else {
		srv.SuggestPeer(seed.IP, seed.Port, nil)
	}

	select {}
}
