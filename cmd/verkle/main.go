package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/crate-crypto/go-ipa/banderwagon"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s <path to block file> <pre-root>\n", os.Args[0])
		os.Exit(-1)
	}

	serializedBlock, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	rootHex, err := hex.DecodeString(os.Args[2])
	if err != nil {
		panic(err)
	}

	var root banderwagon.Element
	root.SetBytes(rootHex)

	var block types.Block
	rlp.DecodeBytes(serializedBlock, &block)

	if len(block.Header().VerkleProof) == 0 {
		panic("missing proof")
	}

	err = trie.DeserializeAndVerifyVerkleProof(block.Header().VerkleProof, &root, block.Header().VerkleKeyVals)
	if err != nil {
		fmt.Printf("error verifying proof: %v\n", err)
	}
}
