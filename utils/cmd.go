package utils

import (
	"encoding/hex"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethminer"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethrpc"
	"github.com/ethereum/eth-go/ethutil"
	"log"
	"time"
)

func DoRpc(ethereum *eth.Ethereum, RpcPort int) {
	var err error
	ethereum.RpcServer, err = ethrpc.NewJsonRpcServer(ethpub.NewPEthereum(ethereum), RpcPort)
	if err != nil {
		log.Println("Could not start RPC interface:", err)
	} else {
		go ethereum.RpcServer.Start()
	}
}

func DoMining(ethereum *eth.Ethereum) {
	// Set Mining status
	ethereum.Mining = true

	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	if len(data) == 0 {
		log.Println("No address found, can't start mining")
		return
	}

	keyRing := ethutil.NewValueFromBytes(data)
	addr := keyRing.Get(0).Bytes()
	pair, _ := ethchain.NewKeyPairFromSec(ethutil.FromHex(hex.EncodeToString(addr)))

	go func() {
		// Give it some time to connect with peers
		time.Sleep(3 * time.Second)

		for ethereum.IsUpToDate() == false {
			time.Sleep(5 * time.Second)
		}
		log.Println("Miner started")

		miner := ethminer.NewDefaultMiner(pair.Address(), ethereum)
		miner.Start()
	}()
}
