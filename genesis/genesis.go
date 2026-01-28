package genesis

import (
	_ "embed"
)

//go:embed mainnet.json
var MainnetGenesis []byte

//go:embed testnet.json
var TestnetGenesis []byte

//go:embed devnet.json
var DevnetGenesis []byte
