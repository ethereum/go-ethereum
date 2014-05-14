package utils

import (
	"encoding/hex"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
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
		data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
		keyRing := ethutil.NewValueFromBytes(data)
		addr := keyRing.Get(0).Bytes()

		pair, _ := ethchain.NewKeyPairFromSec(ethutil.FromHex(hex.EncodeToString(addr)))

		miner := ethminer.NewDefaultMiner(pair.Address(), ethereum)
		miner.Start()

	}()
}
