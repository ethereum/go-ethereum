package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discover/portalwire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/storage/sqlite"
	"github.com/ethereum/go-ethereum/rpc"
)

func main() {
	glogger := log.NewGlogHandler(log.NewTerminalHandler(os.Stderr, true))
	slogVerbosity := log.FromLegacyLevel(5)
	glogger.Verbosity(slogVerbosity)
	log.SetDefault(log.NewLogger(glogger))

	var privateKey *ecdsa.PrivateKey
	var err error
	privateKeyHex := os.Getenv("HIVE_CLIENT_PRIVATE_KEY")
	if privateKeyHex != "" {
		keyBytes, err := hexutil.Decode("0x" + privateKeyHex)
		if err != nil {
			panic(err)
		}
		privateKey, err = crypto.ToECDSA(keyBytes)
		if err != nil {
			panic(err)
		}
	} else {
		privateKey, err = crypto.GenerateKey()
		if err != nil {
			panic(err)
		}
	}

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

	udpPort := os.Getenv("UDP_PORT")

	if udpPort != "" {
		config.ListenAddr = ":" + udpPort
	}
	nodeId := enode.PubkeyToIDV4(&privateKey.PublicKey)
	contentStorage, err := sqlite.NewContentStorage(1000*1000*1000, nodeId, "./")
	if err != nil {
		panic(err)
	}

	contentQueue := make(chan *discover.ContentElement, 50)

	protocol, err := discover.NewPortalProtocol(config, string(portalwire.HistoryNetwork), privateKey, contentStorage, contentQueue)

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

	tcpPort := os.Getenv("TCP_PORT")

	if tcpPort == "" {
		tcpPort = "8545"
	}

	httpServer := &http.Server{
		Addr:    ":" + tcpPort,
		Handler: server,
	}

	httpServer.ListenAndServe()
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
