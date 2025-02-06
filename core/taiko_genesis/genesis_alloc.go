package taiko_genesis

import (
	_ "embed"
)

//go:embed internal_l2a.json
var InternalL2AGenesisAllocJSON []byte

//go:embed internal_l2b.json
var InternalL2BGenesisAllocJSON []byte

//go:embed snaefellsjokull.json
var SnaefellsjokullGenesisAllocJSON []byte

//go:embed askja.json
var AskjaGenesisAllocJSON []byte

//go:embed grimsvotn.json
var GrimsvotnGenesisAllocJSON []byte

//go:embed eldfell.json
var EldfellGenesisAllocJSON []byte

//go:embed jolnir.json
var JolnirGenesisAllocJSON []byte

//go:embed katla.json
var KatlaGenesisAllocJSON []byte

//go:embed hekla.json
var HeklaGenesisAllocJSON []byte

//go:embed mainnet.json
var MainnetGenesisAllocJSON []byte

//go:embed preconf_devnet.json
var PreconfDevnetGenesisAllocJSON []byte
