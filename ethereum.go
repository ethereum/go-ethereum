package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/ethchain-go"
	"github.com/ethereum/ethutil-go"
	"github.com/obscuren/secp256k1-go"
	"log"
	"os"
	"os/signal"
	"runtime"
)

const Debug = true

// Register interrupt handlers so we can stop the ethereum
func RegisterInterupts(s *eth.Ethereum) {
	// Buffered chan of one is enough
	c := make(chan os.Signal, 1)
	// Notify about interrupts for now
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			fmt.Printf("Shutting down (%v) ... \n", sig)

			s.Stop()
		}
	}()
}

func CreateKeyPair(force bool) {
	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	if len(data) == 0 || force {
		log.Println("Generating new address and keypair")

		pub, prv := secp256k1.GenerateKeyPair()

		ethutil.Config.Db.Put([]byte("KeyRing"), prv)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	Init()

	ethchain.InitFees()
	ethutil.ReadConfig()

	// Instantiated a eth stack
	ethereum, err := eth.New(eth.CapDefault, UseUPnP)
	if err != nil {
		log.Println("eth start err:", err)
		return
	}

	if GenAddr {
		fmt.Println("This action overwrites your old private key. Are you sure? (y/n)")

		var r string
		fmt.Scanln(&r)
		for ; ; fmt.Scanln(&r) {
			if r == "n" || r == "y" {
				break
			} else {
				fmt.Println("Yes or no?", r)
			}
		}

		if r == "y" {
			CreateKeyPair(true)
		}
		os.Exit(0)
	} else {
		CreateKeyPair(false)
	}

	if ShowGenesis {
		fmt.Println(ethereum.BlockManager.BlockChain().Genesis())
		os.Exit(0)
	}

	log.Printf("Starting Ethereum v%s\n", ethutil.Config.Ver)

	// Set the max peers
	ethereum.MaxPeers = MaxPeer

	if StartConsole {
		err := os.Mkdir(ethutil.Config.ExecPath, os.ModePerm)
		// Error is OK if the error is ErrExist
		if err != nil && !os.IsExist(err) {
			log.Panic("Unable to create EXECPATH:", err)
		}

		console := NewConsole(ethereum)
		go console.Start()
	}

	RegisterInterupts(ethereum)

	ethereum.Start()

	if StartMining {
		log.Printf("Dev Test Mining started...\n")

		// Fake block mining. It broadcasts a new block every 5 seconds
		go func() {
			pow := &ethchain.EasyPow{}
			addr, _ := hex.DecodeString("82c3b0b72cf62f1a9ce97c64da8072efa28225d8")

			for {
				txs := ethereum.TxPool.Flush()
				// Create a new block which we're going to mine
				block := ethereum.BlockManager.BlockChain().NewBlock(addr, txs)
				// Apply all transactions to the block
				ethereum.BlockManager.ApplyTransactions(block, block.Transactions())

				ethereum.BlockManager.AccumelateRewards(block, block)

				// Search the nonce
				block.Nonce = pow.Search(block)
				err := ethereum.BlockManager.ProcessBlock(block)
				if err != nil {
					log.Println(err)
				} else {
					log.Println("\n+++++++ MINED BLK +++++++\n", ethereum.BlockManager.BlockChain().CurrentBlock)
				}
			}
		}()
	}

	// Wait for shutdown
	ethereum.WaitForShutdown()
}
