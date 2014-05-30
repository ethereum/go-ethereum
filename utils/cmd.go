package utils

import (
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethminer"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethrpc"
	"github.com/ethereum/eth-go/ethutil"
	"time"
)

func DoRpc(ethereum *eth.Ethereum, RpcPort int) {
	var err error
	ethereum.RpcServer, err = ethrpc.NewJsonRpcServer(ethpub.NewPEthereum(ethereum), RpcPort)
	if err != nil {
		ethutil.Config.Log.Infoln("Could not start RPC interface:", err)
	} else {
		go ethereum.RpcServer.Start()
	}
}

var miner ethminer.Miner

func DoMining(ethereum *eth.Ethereum) {
	// Set Mining status
	ethereum.Mining = true

	if ethutil.GetKeyRing().Len() == 0 {
		ethutil.Config.Log.Infoln("No address found, can't start mining")
		return
	}
	keyPair := ethutil.GetKeyRing().Get(0)
	addr := keyPair.Address()

	go func() {
		miner = ethminer.NewDefaultMiner(addr, ethereum)

		// Give it some time to connect with peers
		time.Sleep(3 * time.Second)

		for ethereum.IsUpToDate() == false {
			time.Sleep(5 * time.Second)
		}

		ethutil.Config.Log.Infoln("Miner started")

		miner := ethminer.NewDefaultMiner(addr, ethereum)
		miner.Start()
	}()
}

func StopMining(ethereum *eth.Ethereum) bool {
	if ethereum.Mining {
		miner.Stop()

		ethutil.Config.Log.Infoln("Miner stopped")

		ethereum.Mining = false

		return true
	}

	return false
}

func StartMining(ethereum *eth.Ethereum) bool {
	if !ethereum.Mining {
		DoMining(ethereum)

		return true
	}

	return false
}
