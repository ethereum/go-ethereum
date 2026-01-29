// XDPoS Engines - Common definitions for geth 1.17
package engines

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	ExtraVanity = 32 // Bytes reserved for signer vanity
	ExtraSeal   = 65 // Bytes reserved for signer seal
)

// SignerFn is a signer callback function
type SignerFn func(common.Address, []byte) ([]byte, error)
