package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func TestPermissionDiscovery(t *testing.T) {
	prvKeys := []*ecdsa.PrivateKey{newkey(), newkey(), newkey(), newkey()}
	validatedPubkeys := []*ecdsa.PublicKey{}

	// Last index is non validated key
	for i := 0; i < len(prvKeys)-1; i++ {
		pubkey := prvKeys[i].PublicKey
		validatedPubkeys = append(validatedPubkeys, &pubkey)
	}

	srvs := []*Server{}

	bootSrv := &Server{
		Config: Config{
			PrivateKey: prvKeys[0],
			MaxPeers:   50,
			// Logger:     testlog.Logger(t, log.LvlTrace).New("bootServer", 0),
			ListenAddr: "0.0.0.0:0",
			Protocols: []Protocol{{
				Name:    "Test",
				Version: 1,
				Length:  1,
				Run: func(peer *Peer, rw MsgReadWriter) error {
					fmt.Println("Bootnode", "Peer connected", peer.RemoteAddr(), peer.ID())
					for {
					}
				},
			}},
		},
		ValidatedPubkeys: validatedPubkeys,
	}

	bootSrv.Start()
	srvs = append(srvs, bootSrv)

	bootNode, err := enode.ParseV4(bootSrv.NodeInfo().Enode)
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("BootstrapNode", bootNode.ID())

	for i := 1; i < len(prvKeys); i++ {
		srv := &Server{
			Config: Config{
				PrivateKey: prvKeys[i],
				MaxPeers:   50,
				// Logger:         testlog.Logger(t, log.LvlTrace).New("Server", i),
				BootstrapNodes: []*enode.Node{bootNode},
				ListenAddr:     "0.0.0.0:0",
				Protocols: []Protocol{{
					Name:    "Test",
					Version: 1,
					Length:  1,
					Run: func(peer *Peer, rw MsgReadWriter) error {
						fmt.Println("Server", "Peer connected", peer.RemoteAddr(), peer.ID())
						for {
						}
					},
				}},
			},
		}
		err = srv.Start()
		if err != nil {
			t.Fatal(err)
		}
		srvs = append(srvs, srv)
	}

	expectedPeerPerNode := len(validatedPubkeys) - 1 // exclude self
	expectedFullPeerCount := expectedPeerPerNode * len(validatedPubkeys)

	for {
		fullPeerCount := 0
		for _, server := range srvs {
			fullPeerCount += server.PeerCount()
		}

		if fullPeerCount == expectedFullPeerCount {
			break
		}

		fmt.Println("Sleep 5 seconds ...")
		time.Sleep(5 * time.Second)
	}

	for _, server := range srvs {
		peers := server.Peers()
		for _, peer := range peers {
			peerPubkey := peer.Node().Pubkey()
			peerPubkeyByte := crypto.FromECDSAPub(peerPubkey)

			validated := false
			for _, validatedPubkey := range validatedPubkeys {
				validatedPubkeyByte := crypto.FromECDSAPub(validatedPubkey)
				if bytes.Equal(validatedPubkeyByte, peerPubkeyByte) {
					validated = true
				}
			}

			if !validated {
				t.Error(server.NodeInfo().ID, "Node connected to non-whitelisted peer", peer.Node().ID())
			}
		}
	}

}
