package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/storage/sqlite"
	"github.com/ethereum/go-ethereum/rpc"
)

func main() {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	// testStr := "enr:-Iq4QM89fOvgXRfNo429L4vD_qn3GEAQqelOQzMAJCBZs7oZZ3t6F07P-J2iNUMtyeDcvTzBqP4lJD0EllJdht7GOYqGAY1GIhc5gmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQM0MbiqlCBmJu-8T2tC4z9KqlcyB6HkdRsASowWTVrQAoN1ZHCCIyg"

	// if err != nil {
	// 	panic(err)
	// }
	// record := new(enr.Record)
	// err = rlp.DecodeBytes([]byte(testStr), record)
	// if err != nil {
	// 	panic(err)
	// }

	config := discover.DefaultPortalProtocolConfig()

	bootNodeStr := os.Getenv("HIVE_BOOTNODE")
	if bootNodeStr != "" {
		bootNode := new(enode.Node)
		err = bootNode.UnmarshalText([]byte(bootNodeStr))
		if err != nil {
			panic(err)
		}
		config.BootstrapNodes = append(config.BootstrapNodes, bootNode)
	}
	nodeId := enode.PubkeyToIDV4(&privateKey.PublicKey)
	contentStorage, err := sqlite.NewContentStorage(1000*1000*1000, nodeId, "./")
	if err != nil {
		panic(err)
	}

	contentQueue := make(chan *discover.ContentElement, 50)

	protocol, err := discover.NewPortalProtocol(config, "history", privateKey, contentStorage, contentQueue)

	if err != nil {
		panic(err)
	}

	err = protocol.Start()
	if err != nil {
		panic(err)
	}

	disv5 := discover.NewAPI(protocol.DiscV5)
	portal := discover.NewPortalAPI(protocol)

	server := rpc.NewServer()
	server.RegisterName("discv5", disv5)
	server.RegisterName("portal", portal)

	httpServer := &http.Server{
		Addr:    ":8545",
		Handler: server,
	}
	fmt.Printf("before http")
	httpServer.ListenAndServe()
	fmt.Printf("success")
}

func ReadKeyFromFile(name string) (*ecdsa.PrivateKey, error) {
	keyBytes, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}
