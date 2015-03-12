package core

import "github.com/ethereum/go-ethereum/core/types"

// TxPreEvent is posted when a transaction enters the transaction pool.
type TxPreEvent struct{ Tx *types.Transaction }

// TxPostEvent is posted when a transaction has been processed.
type TxPostEvent struct{ Tx *types.Transaction }

// NewBlockEvent is posted when a block has been imported.
type NewBlockEvent struct{ Block *types.Block }

// NewMinedBlockEvent is posted when a block has been imported.
type NewMinedBlockEvent struct{ Block *types.Block }

// ChainSplit is posted when a new head is detected
type ChainSplitEvent struct{ Block *types.Block }

type ChainEvent struct{ Block *types.Block }

type ChainSideEvent struct{ Block *types.Block }

type ChainHeadEvent struct{ Block *types.Block }

// Mining operation events
type StartMining struct{}
type TopMining struct{}
