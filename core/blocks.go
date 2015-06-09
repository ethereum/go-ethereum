package core

import "github.com/ethereum/go-ethereum/common"

// Set of manually tracked bad hashes (usually hard forks)
var BadHashes = map[common.Hash]bool{
	common.HexToHash("f269c503aed286caaa0d114d6a5320e70abbc2febe37953207e76a2873f2ba79"): true,
	common.HexToHash("38f5bbbffd74804820ffa4bab0cd540e9de229725afb98c1a7e57936f4a714bc"): true,
	common.HexToHash("7064455b364775a16afbdecd75370e912c6e2879f202eda85b9beae547fff3ac"): true,
	common.HexToHash("5b7c80070a6eff35f3eb3181edb023465c776d40af2885571e1bc4689f3a44d8"): true,
}
