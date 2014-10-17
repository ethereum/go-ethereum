package ethchain

type TxEvent struct {
	Type int // TxPre || TxPost
	Tx   *Transaction
}

type NewBlockEvent struct {
	Block *Block
}
