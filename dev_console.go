package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
	_ "math/big"
	"os"
	"strings"
)

type Console struct {
	db       *ethdb.MemDatabase
	trie     *ethutil.Trie
	ethereum *eth.Ethereum
}

func NewConsole(s *eth.Ethereum) *Console {
	db, _ := ethdb.NewMemDatabase()
	trie := ethutil.NewTrie(db, "")

	return &Console{db: db, trie: trie, ethereum: s}
}

func (i *Console) ValidateInput(action string, argumentLength int) error {
	err := false
	var expArgCount int

	switch {
	case action == "update" && argumentLength != 2:
		err = true
		expArgCount = 2
	case action == "get" && argumentLength != 1:
		err = true
		expArgCount = 1
	case action == "dag" && argumentLength != 2:
		err = true
		expArgCount = 2
	case action == "decode" && argumentLength != 1:
		err = true
		expArgCount = 1
	case action == "encode" && argumentLength != 1:
		err = true
		expArgCount = 1
	case action == "gettx" && argumentLength != 1:
		err = true
		expArgCount = 1
	case action == "tx" && argumentLength != 2:
		err = true
		expArgCount = 2
	case action == "getaddr" && argumentLength != 1:
		err = true
		expArgCount = 1
	case action == "contract" && argumentLength != 1:
		err = true
		expArgCount = 1
	case action == "say" && argumentLength != 1:
		err = true
		expArgCount = 1
	case action == "addp" && argumentLength != 1:
		err = true
		expArgCount = 1
	case action == "block" && argumentLength != 1:
		err = true
		expArgCount = 1
	}

	if err {
		return errors.New(fmt.Sprintf("'%s' requires %d args, got %d", action, expArgCount, argumentLength))
	} else {
		return nil
	}
}

func (i *Console) Editor() []string {
	var buff bytes.Buffer
	for {
		reader := bufio.NewReader(os.Stdin)
		str, _, err := reader.ReadLine()
		if len(str) > 0 {
			buff.Write(str)
			buff.WriteString("\n")
		}

		if err != nil && err.Error() == "EOF" {
			break
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(buff.String()))
	scanner.Split(bufio.ScanLines)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

func (i *Console) PrintRoot() {
	root := ethutil.NewValue(i.trie.Root)
	if len(root.Bytes()) != 0 {
		fmt.Println(hex.EncodeToString(root.Bytes()))
	} else {
		fmt.Println(i.trie.Root)
	}
}

func (i *Console) ParseInput(input string) bool {
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(bufio.ScanWords)

	count := 0
	var tokens []string
	for scanner.Scan() {
		count++
		tokens = append(tokens, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading input:", err)
	}

	if len(tokens) == 0 {
		return true
	}

	err := i.ValidateInput(tokens[0], count-1)
	if err != nil {
		fmt.Println(err)
	} else {
		switch tokens[0] {
		case "update":
			i.trie.Update(tokens[1], tokens[2])

			i.PrintRoot()
		case "get":
			fmt.Println(i.trie.Get(tokens[1]))
		case "root":
			i.PrintRoot()
		case "rawroot":
			fmt.Println(i.trie.Root)
		case "print":
			i.db.Print()
		case "dag":
			fmt.Println(ethchain.DaggerVerify(ethutil.Big(tokens[1]), // hash
				ethutil.BigPow(2, 36),   // diff
				ethutil.Big(tokens[2]))) // nonce
		case "decode":
			value := ethutil.NewValueFromBytes([]byte(tokens[1]))
			fmt.Println(value)
		case "getaddr":
			encoded, _ := hex.DecodeString(tokens[1])
			addr := i.ethereum.BlockChain().CurrentBlock.State().GetAccount(encoded)
			fmt.Println("addr:", addr)
		case "block":
			encoded, _ := hex.DecodeString(tokens[1])
			block := i.ethereum.BlockChain().GetBlock(encoded)
			info := block.BlockInfo()
			fmt.Printf("++++++++++ #%d ++++++++++\n%v\n", info.Number, block)
		case "say":
			i.ethereum.Broadcast(ethwire.MsgTalkTy, []interface{}{tokens[1]})
		case "addp":
			i.ethereum.ConnectToPeer(tokens[1])
		case "pcount":
			fmt.Println("peers:", i.ethereum.Peers().Len())
		case "encode":
			fmt.Printf("%q\n", ethutil.Encode(tokens[1]))
		case "tx":
			recipient, err := hex.DecodeString(tokens[1])
			if err != nil {
				fmt.Println("recipient err:", err)
			} else {
				tx := ethchain.NewTransaction(recipient, ethutil.Big(tokens[2]), []string{""})

				key := ethutil.Config.Db.GetKeys()[0]
				tx.Sign(key.PrivateKey)
				i.ethereum.TxPool().QueueTransaction(tx)

				fmt.Printf("%x\n", tx.Hash())
			}
		case "gettx":
			addr, _ := hex.DecodeString(tokens[1])
			data, _ := ethutil.Config.Db.Get(addr)
			if len(data) != 0 {
				decoder := ethutil.NewValueFromBytes(data)
				fmt.Println(decoder)
			} else {
				fmt.Println("gettx: tx not found")
			}
		case "contract":
			fmt.Println("Contract editor (Ctrl-D = done)")
			code := ethchain.Compile(i.Editor())

			contract := ethchain.NewTransaction(ethchain.ContractAddr, ethutil.Big(tokens[1]), code)

			key := ethutil.Config.Db.GetKeys()[0]
			contract.Sign(key.PrivateKey)

			i.ethereum.TxPool().QueueTransaction(contract)

			fmt.Printf("%x\n", contract.Hash()[12:])
		case "exit", "quit", "q":
			return false
		case "help":
			fmt.Printf("COMMANDS:\n" +
				"\033[1m= DB =\033[0m\n" +
				"update KEY VALUE - Updates/Creates a new value for the given key\n" +
				"get KEY - Retrieves the given key\n" +
				"root - Prints the hex encoded merkle root\n" +
				"rawroot - Prints the raw merkle root\n" +
				"block HASH - Prints the block\n" +
				"getaddr ADDR - Prints the account associated with the address\n" +
				"\033[1m= Dagger =\033[0m\n" +
				"dag HASH NONCE - Verifies a nonce with the given hash with dagger\n" +
				"\033[1m= Encoding =\033[0m\n" +
				"decode STR\n" +
				"encode STR\n" +
				"\033[1m= Other =\033[0m\n" +
				"addp HOST:PORT\n" +
				"tx TO AMOUNT\n" +
				"contract AMOUNT\n")

		default:
			fmt.Println("Unknown command:", tokens[0])
		}
	}

	return true
}

func (i *Console) Start() {
	fmt.Printf("Eth Console. Type (help) for help\n")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("eth >>> ")
		str, _, err := reader.ReadLine()
		if err != nil {
			fmt.Println("Error reading input", err)
		} else {
			if !i.ParseInput(string(str)) {
				return
			}
		}
	}
}
