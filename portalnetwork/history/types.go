package history

type HeaderRecord struct {
	BlockHash       []byte `ssz-size:"32"`
	TotalDifficulty []byte `ssz-size:"32"`
}

type BlockBodyLegacy struct {
	Transactions [][]byte `ssz-max:"16384,16777216"`
	Uncles       []byte   `ssz-max:"131072"`
}

type PortalBlockBodyShanghai struct {
	Transactions [][]byte `ssz-max:"16384,16777216"`
	Uncles       []byte   `ssz-max:"131072"`
	Withdrawals  [][]byte `ssz-max:"16,192"`
}

type BlockHeaderWithProof struct {
	Header []byte `ssz-max:"8192"`
	Proof  []byte `ssz-max:"512"`
}

type SSZProof struct {
	Leaf      []byte   `ssz-size:"32"`
	Witnesses [][]byte `ssz-max:"65536,32" ssz-size:"?,32"`
}

type MasterAccumulator struct {
	HistoricalEpochs [][]byte `ssz-max:"1897,32" ssz-size:"?,32"`
}
