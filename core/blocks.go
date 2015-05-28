package core

import "github.com/ethereum/go-ethereum/common"

// Set of manually tracked bad hashes (usually hard forks)
var BadHashes = map[common.Hash]bool{
	common.HexToHash("f269c503aed286caaa0d114d6a5320e70abbc2febe37953207e76a2873f2ba79"): true,
	common.HexToHash("38f5bbbffd74804820ffa4bab0cd540e9de229725afb98c1a7e57936f4a714bc"): true,
}
