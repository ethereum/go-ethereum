package main

import (
	"bytes"
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
		pair := &ethutil.Key{PrivateKey: prv, PublicKey: pub}
		ethutil.Config.Db.Put([]byte("KeyRing"), pair.RlpEncode())

		fmt.Printf(`
Generating new address and keypair.
Please keep your keys somewhere save.

++++++++++++++++ KeyRing +++++++++++++++++++
addr: %x
prvk: %x
pubk: %x
++++++++++++++++++++++++++++++++++++++++++++

`, pair.Address(), prv, pub)

	}
}

func ImportPrivateKey(prvKey string) {
	key := ethutil.FromHex(prvKey)
	msg := []byte("tmp")
	// Couldn't think of a better way to get the pub key
	sig, _ := secp256k1.Sign(msg, key)
	pub, _ := secp256k1.RecoverPubkey(msg, sig)
	pair := &ethutil.Key{PrivateKey: key, PublicKey: pub}
	ethutil.Config.Db.Put([]byte("KeyRing"), pair.RlpEncode())

	fmt.Printf(`
Importing private key

++++++++++++++++ KeyRing +++++++++++++++++++
addr: %x
prvk: %x
pubk: %x
++++++++++++++++++++++++++++++++++++++++++++

`, pair.Address(), key, pub)
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
	if err != nil {
		log.Println("eth start err:", err)
		return
	}
	ethereum.Port = OutboundPort

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
				os.Exit(0)
			}
		} else {
			CreateKeyPair(false)
		}
	}

	if ExportKey {
		key := ethutil.Config.Db.GetKeys()[0]
		fmt.Printf("%x\n", key.PrivateKey)
		os.Exit(0)
	}

	if ShowGenesis {
		fmt.Println(ethereum.BlockChain().Genesis())
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

		minerChan := make(chan ethutil.React, 5)
		ethereum.Reactor().Subscribe("newBlock", minerChan)
		ethereum.Reactor().Subscribe("newTx", minerChan)

		minerChan2 := make(chan ethutil.React, 5)
		ethereum.Reactor().Subscribe("newBlock", minerChan2)
		ethereum.Reactor().Subscribe("newTx", minerChan2)

		ethereum.StateManager().PrepareMiningState()

		if StartMining {
			log.Printf("Miner started\n")

			// Fake block mining. It broadcasts a new block every 5 seconds
			go func() {
				pow := &ethchain.EasyPow{}
				data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
				keyRing := ethutil.NewValueFromBytes(data)
				addr := keyRing.Get(1).Bytes()
				txs := ethereum.TxPool().Flush()
				block := ethereum.BlockChain().NewBlock(addr, txs)

				for {
					select {
					case chanMessage := <-minerChan:
						log.Println("REACTOR: Got new block")

						if block, ok := chanMessage.Resource.(*ethchain.Block); ok {
							if bytes.Compare(ethereum.BlockChain().CurrentBlock.Hash(), block.Hash()) == 0 {
								// TODO: Perhaps continue mining to get some uncle rewards
								log.Println("New top block found resetting state")
								// Reapplies the latest block to the mining state, thus resetting
								ethereum.StateManager().PrepareMiningState()
								block = ethereum.BlockChain().NewBlock(addr, txs)
								log.Println("Block set")
							} else {
								if bytes.Compare(block.PrevHash, ethereum.BlockChain().CurrentBlock.PrevHash) == 0 {
									log.Println("HELLO UNCLE")
									// TODO: Add uncle to block
								}
							}
						}

						if tx, ok := chanMessage.Resource.(*ethchain.Transaction); ok {
							log.Println("REACTOR: Got new transaction", tx)
							found := false
							for _, ctx := range txs {
								if found = bytes.Compare(ctx.Hash(), tx.Hash()) == 0; found {
									break
								}

							}
							if found == false {
								log.Println("We did not know about this transaction, adding")
								txs = append(txs, tx)
							} else {
								log.Println("We already had this transaction, ignoring")
							}
						}
						log.Println("Sending block reset")
						// Start mining over
						log.Println("Block reset done")
					default:
						// Create a new block which we're going to mine
						log.Println("Mining on block. Includes", len(txs), "transactions")

						// Apply all transactions to the block
						ethereum.StateManager().ApplyTransactions(block, txs)
						ethereum.StateManager().AccumelateRewards(block, block)

						// Search the nonce
						block.Nonce = pow.Search(block, minerChan2)
						if block.Nonce != nil {
							ethereum.Broadcast(ethwire.MsgBlockTy, []interface{}{block.Value().Val})
							err := ethereum.StateManager().ProcessBlock(block)
							if err != nil {
								log.Println(err)
							} else {
								//log.Println("\n+++++++ MINED BLK +++++++\n", ethereum.BlockChain().CurrentBlock)
								log.Printf("🔨  Mined block %x\n", block.Hash())
								block = ethereum.BlockChain().NewBlock(addr, txs)
							}
						}
					}
				}
			}()
		}

		// Wait for shutdown
		ethereum.WaitForShutdown()
	}
}
