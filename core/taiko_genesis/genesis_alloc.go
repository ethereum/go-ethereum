package taiko_genesis

import (
	_ "embed"
)

//go:embed mainnet.json
var MainnetGenesisAllocJSON []byte

//go:embed alpha-1.json
var Alpha1GenesisAllocJSON []byte

//go:embed alpha-2.json
var Alpha2GenesisAllocJSON []byte
