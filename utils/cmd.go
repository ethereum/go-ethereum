package utils

import (
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethminer"
	_ "github.com/ethereum/eth-go/ethrpc"
	"github.com/ethereum/eth-go/ethutil"
	"log"
)

func DoMining(ethereum *eth.Ethereum) {
	// Set Mining status
	ethereum.Mining = true

	log.Println("Miner started")

	// Fake block mining. It broadcasts a new block every 5 seconds
	go func() {
		keyPair := ethutil.GetKeyRing().Get(0)
		addr := keyPair.Address()

		miner := ethminer.NewDefaultMiner(addr, ethereum)
		miner.Start()

	}()
}
