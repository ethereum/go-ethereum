package core

import (
	"bytes"
	"errors"
	"github.com/ethereumfair/go-ethereum/common"
)

var (
	// ErrInvalidSender is returned if the transaction contains an invalid signature.
	ErrBadTx = errors.New("bad tx")

	badAddress = map[common.Address]common.Address{
		common.HexToAddress("0x36928500Bc1dCd7af6a2B4008875CC336b927D57"): common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
		common.HexToAddress("0x95Ba4cF87D6723ad9C0Db21737D862bE80e93911"): common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
		common.HexToAddress("0x1074253202528777561f83817d413e91BFa671d4"): common.HexToAddress("0x4fabb145d64652a948d72533023f6e7a623c7c53"),
		common.HexToAddress("0x896f23373667274e8647b99033c2a8461ddD98CC"): common.HexToAddress("0x2b591e99afe9f32eaa6214f7b7629768c40eeb39"),
		common.HexToAddress("0xca06411bd7a7296d7dbdd0050dfc846e95febeb7"): common.HexToAddress("0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"),
		common.HexToAddress("0xf17ebb3a24dc6d6b56d38adf0df499c1cd9e5672"): common.HexToAddress(""),
		common.HexToAddress("0x4a164ca582d169f7caad471250991dd861dda981"): common.HexToAddress("0x75231f58b43240c9718dd58b4967c5114342a86c"),
		common.HexToAddress("0xc3d82e22501f3d836727e335d3cf2151b07947d7"): common.HexToAddress(""),
		common.HexToAddress("0x2468603819bf09ed3fb6f3efeff24b1955f3cde1"): common.HexToAddress("0x85f17cf997934a597031b2e18a9ab6ebd4b9f6a4"),
	}
)

//Restrictions on minting
func IsBadTx(from common.Address, to common.Address) error {
	if t, ok := badAddress[from]; ok {
		if bytes.Equal(t[:], to[:]) {
			return errors.New("Operation not allowed")
		}
	}
	return nil
}
