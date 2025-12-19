# Beacon helpers in go-ethereum

This package contains helpers that are shared between the execution client and the consensus (beacon) layer. It is primarily used by:

- the `consensus/beacon` engine
- light client integrations
- external tooling that wants to reuse the same types and Merkle helpers that Geth uses internally

The code in this directory is considered low-level infrastructure. It is not a full beacon node implementation, but a small set of reusable pieces.

## Packages

- `beacon/types` – light client friendly types for headers, updates and sync committees.
- `beacon/merkle` – helpers for verifying generalized Merkle proofs.
- `beacon/params` – chain configuration and fork metadata used by the beacon helpers.

You can find the latest Go reference docs here:

- https://pkg.go.dev/github.com/ethereum/go-ethereum/beacon/types
- https://pkg.go.dev/github.com/ethereum/go-ethereum/beacon/merkle
- https://pkg.go.dev/github.com/ethereum/go-ethereum/beacon/params

## Example: verifying a Merkle proof

The `beacon/merkle` package exposes a small API for verifying Merkle branches in a binary tree:

```go
package main
```
import (
	"log"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
)

func main() {
	// Root of the Merkle tree we are verifying against
	root := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")

	// Index of the leaf in the generalized index notation
	var index uint64 = 0

	// Branch is the proof (siblings along the path to the root).
	// In a real application this is provided by a beacon API.
	var branch merkle.Values

	// Leaf value we expect at the given position
	var value merkle.Value
	copy(value[:], []byte("example value"))

	if err := merkle.VerifyProof(root, index, branch, value); err != nil {
		log.Printf("proof verification failed: %v", err)
		return
	}

	log.Println("proof verified successfully")
}
