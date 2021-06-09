package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	_ "os"
	_ "strconv"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

// state.Account, just FYI
// type Account struct {
// 	Nonce    uint64
// 	Balance  *big.Int
// 	Root     common.Hash // merkle root of the storage trie
// 	CodeHash []byte
// }

// emptyRoot is the known root hash of an empty trie.
var emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

// emptyCode is the known hash of the empty EVM bytecode.
var emptyCode = crypto.Keccak256(nil)
var emptyCodeHash = crypto.Keccak256(nil)

func randomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

func main() {

	// make trie
	normTrie := trie.NewEmpty()
	secureTrie := trie.NewEmptySecure()

	sizeCheckEpoch := 1
	accountsNum := 4
	emptyStateDB := &state.StateDB{}
	emptyAccount := state.Account{}
	trieCommitEpoch := 1

	// create trie size log file
	// normTrieSizeLog, _ := os.Create("./normTrieSizeLog" + "_" + strconv.Itoa(accountsNum) + "_" + strconv.Itoa(sizeCheckEpoch) + ".txt")
	// defer normTrieSizeLog.Close()
	// secureTrieSizeLog, _ := os.Create("./secureTrieSizeLog" + "_" + strconv.Itoa(accountsNum) + "_" + strconv.Itoa(sizeCheckEpoch) + ".txt")
	// defer secureTrieSizeLog.Close()

	for i := 1; i <= accountsNum; i++ {

		// make random address
		//randHex := randomHex(20)
		//fmt.Println("random hex string:", randHex)

		// make incremental hex
		randHex := fmt.Sprintf("%x", i) // make int as hex string
		//fmt.Println("address hex string:", randHex)

		randAddr := common.HexToAddress(randHex)
		fmt.Println("insert account addr:", randAddr.Hex())

		// set account info
		emptyAccount.Nonce = uint64(0)
		emptyAccount.Balance = big.NewInt(1)
		emptyAccount.Root = emptyRoot
		emptyAccount.CodeHash = emptyCodeHash
		
		// encoding value
		emptyStateObject := state.NewObject(emptyStateDB, randAddr, emptyAccount)
		data, _ := rlp.EncodeToBytes(emptyStateObject)

		// insert account into trie
		normTrie.TryUpdate(randAddr[:], data)
		secureTrie.TryUpdate(randAddr[:], data)

		// commit trie changes
		if i%trieCommitEpoch == 0 {
			fmt.Println("commit norm trie")
			normTrie.Commit(nil)
			normTrie.MyCommit()

			fmt.Println("\ncommit secure trie")
			secureTrie.Commit(nil)
			secureTrie.MyCommit()
		}

		// write trie storage size
		if i%sizeCheckEpoch == 0 {
			fmt.Println("# of accounts:", i)
			fmt.Println("trie size:", normTrie.Size(), "/ secure trie size:", secureTrie.Trie().Size())

			// sizeLog := strconv.Itoa(i) + "\t" + strconv.FormatInt(normTrie.Size().Int(), 10) + "\n"
			// normTrieSizeLog.WriteString(sizeLog)

			// sizeLog = strconv.Itoa(i) + "\t" + strconv.FormatInt(secureTrie.Trie().Size().Int(), 10) + "\n"
			// secureTrieSizeLog.WriteString(sizeLog)
		}

		fmt.Println("\nprint norm trie")
		normTrie.Print()
		fmt.Println("\nprint secure trie")
		secureTrie.Trie().Print()
		fmt.Println("\n\n\n\n\n")
	}

	// print trie nodes
	// fmt.Println("\nprint norm trie")
	// normTrie.Print()
	// fmt.Println("\nprint secure trie")
	// secureTrie.Trie().Print()


}
