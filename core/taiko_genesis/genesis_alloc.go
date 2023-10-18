package taiko_genesis

import (
	_ "embed"
)

//go:embed internal.json
var InternalGenesisAllocJSON []byte

//go:embed internal_l3.json
var InternalL3GenesisAllocJSON []byte

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
