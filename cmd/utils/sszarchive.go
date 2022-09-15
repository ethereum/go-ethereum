package utils

import "github.com/ethereum/go-ethereum/core/types/sszcodec"

const (
	Version   = 0
	MaxBlocks = 1000000
)

type ArchiveHeader struct {
	Version         uint64
	HeadBlockNumber uint64
	BlockCount      uint32
}
type ArchiveBody struct {
	Blocks []*sszcodec.Block `ssz-max:"1000000"`
}
