package utils

import (
	"encoding/hex"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethminer"
	_ "github.com/ethereum/eth-go/ethrpc"
	"github.com/ethereum/eth-go/ethutil"
	"log"
	"time"
)

func DoMining(ethereum *eth.Ethereum) {
	// Set Mining status
	ethereum.Mining = true

	go func() {
		// Give it some time to connect with peers
		time.Sleep(3 * time.Second)

		for ethereum.IsUpToDate() == false {
			time.Sleep(5 * time.Second)
		}
		log.Println("Miner started")

		data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
		keyRing := ethutil.NewValueFromBytes(data)
		addr := keyRing.Get(0).Bytes()
		pair, _ := ethchain.NewKeyPairFromSec(ethutil.FromHex(hex.EncodeToString(addr)))
		miner := ethminer.NewDefaultMiner(pair.Address(), ethereum)
		miner.Start()
	}()
}
