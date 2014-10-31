package chain

// TxPreEvent is posted when a transaction enters the transaction pool.
type TxPreEvent struct{ Tx *Transaction }

// TxPostEvent is posted when a transaction has been processed.
type TxPostEvent struct{ Tx *Transaction }

// NewBlockEvent is posted when a block has been imported.
type NewBlockEvent struct{ Block *Block }
