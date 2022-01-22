package ethtest

import "github.com/ethereum/go-ethereum/eth/protocols/snap"

// GetAccountRange represents an account range query.
type GetAccountRange snap.GetAccountRangePacket

func (g GetAccountRange) Code() int { return 33 }

type AccountRange snap.AccountRangePacket

func (g AccountRange) Code() int { return 34 }
