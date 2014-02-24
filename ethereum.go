package main

import (
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
	"github.com/ethereum/go-ethereum/ui"
	"github.com/niemeyer/qml"
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
		pub, prv := secp256k1.GenerateKeyPair()
		addr := ethutil.Sha3Bin(pub[1:])[12:]

		fmt.Printf(`
Generating new address and keypair.
Please keep your keys somewhere save.

++++++++++++++++ KeyRing +++++++++++++++++++
addr: %x
prvk: %x
pubk: %x
++++++++++++++++++++++++++++++++++++++++++++

`, addr, prv, pub)

		keyRing := ethutil.NewValue([]interface{}{prv, addr, pub[1:]})
		ethutil.Config.Db.Put([]byte("KeyRing"), keyRing.Encode())
	}
}

func ImportPrivateKey(prvKey string) {
	key := ethutil.FromHex(prvKey)
	msg := []byte("tmp")
	// Couldn't think of a better way to get the pub key
	sig, _ := secp256k1.Sign(msg, key)
	pub, _ := secp256k1.RecoverPubkey(msg, sig)
	addr := ethutil.Sha3Bin(pub[1:])[12:]

	fmt.Printf(`
Importing private key

++++++++++++++++ KeyRing +++++++++++++++++++
addr: %x
prvk: %x
pubk: %x
++++++++++++++++++++++++++++++++++++++++++++

`, addr, key, pub)

	keyRing := ethutil.NewValue([]interface{}{key, addr, pub[1:]})
	ethutil.Config.Db.Put([]byte("KeyRing"), keyRing.Encode())
}

func main() {
	Init()

	// Qt has to be initialized in the main thread or it will throw errors
	// It has to be called BEFORE setting the maximum procs.
	if UseGui {
		qml.Init(nil)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	ethchain.InitFees()
	ethutil.ReadConfig(DataDir)
	ethutil.Config.Seed = UseSeed

	// Instantiated a eth stack
	ethereum, err := eth.New(eth.CapDefault, UseUPnP)
	ethereum.Port = OutboundPort
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
				fmt.Printf("Yes or no?", r)
			}
		}

		if r == "y" {
			CreateKeyPair(true)
		}
		os.Exit(0)
	} else {
		if len(ImportKey) > 0 {
			fmt.Println("This action overwrites your old private key. Are you sure? (y/n)")
			var r string
			fmt.Scanln(&r)
			for ; ; fmt.Scanln(&r) {
				if r == "n" || r == "y" {
					break
				} else {
					fmt.Printf("Yes or no?", r)
				}
			}

			if r == "y" {
				ImportPrivateKey(ImportKey)
			}
		} else {
			CreateKeyPair(false)
		}
	}

	if ExportKey {
		data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
		keyRing := ethutil.NewValueFromBytes(data)
		fmt.Printf("%x\n", keyRing.Get(0).Bytes())
		os.Exit(0)
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

	if UseGui {
		gui := ethui.New(ethereum)
		gui.Start()
		//ethereum.Stop()
	} else {
		RegisterInterupts(ethereum)
		ethereum.Start()

		if StartMining {
			log.Printf("Miner started\n")

			// Fake block mining. It broadcasts a new block every 5 seconds
			go func() {
				pow := &ethchain.EasyPow{}
				data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
				keyRing := ethutil.NewValueFromBytes(data)
				addr := keyRing.Get(1).Bytes()

				for {
					txs := ethereum.TxPool.Flush()
					// Create a new block which we're going to mine
					block := ethereum.BlockManager.BlockChain().NewBlock(addr, txs)
					// Apply all transactions to the block
					ethereum.BlockManager.ApplyTransactions(block, block.Transactions())

					ethereum.BlockManager.AccumelateRewards(block, block)

					// Search the nonce
					block.Nonce = pow.Search(block)
					ethereum.Broadcast(ethwire.MsgBlockTy, []interface{}{block.Value().Val})
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
}
