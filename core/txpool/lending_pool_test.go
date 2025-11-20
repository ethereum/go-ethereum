package txpool

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/XinFinOrg/XDPoSChain/XDCxlending/lendingstate"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	"golang.org/x/crypto/sha3"
)

type LendingMsg struct {
	AccountNonce    uint64         `json:"nonce"    gencodec:"required"`
	Quantity        *big.Int       `json:"quantity,omitempty"`
	RelayerAddress  common.Address `json:"relayerAddress,omitempty"`
	UserAddress     common.Address `json:"userAddress,omitempty"`
	CollateralToken common.Address `json:"collateralToken,omitempty"`
	AutoTopUp       bool           `json:"autoTopUp,omitempty"`
	LendingToken    common.Address `json:"lendingToken,omitempty"`
	Term            uint64         `json:"term,omitempty"`
	Interest        uint64         `json:"interest,omitempty"`
	Status          string         `json:"status,omitempty"`
	Side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	LendingId       uint64         `json:"lendingId,omitempty"`
	LendingTradeId  uint64         `json:"tradeId,omitempty"`
	ExtraData       string         `json:"extraData,omitempty"`
	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash common.Hash `json:"hash" rlp:"-"`
}

func getLendingNonce(userAddress common.Address) (uint64, error) {
	rpcClient, err := rpc.DialHTTP("http://127.0.0.1:8501")
	if err != nil {
		return 0, err
	}
	defer rpcClient.Close()
	var result interface{}
	err = rpcClient.Call(&result, "XDCx_getLendingOrderCount", userAddress)
	if err != nil {
		return 0, err
	}
	s := result.(string)
	s = strings.TrimPrefix(s, "0x")
	n, err := strconv.ParseUint(s, 16, 32)
	return uint64(n), err
}

func (l *LendingMsg) computeHash() common.Hash {
	borrowing := l.Side == lendingstate.Borrowing
	sha := sha3.NewLegacyKeccak256()
	if l.Type == lendingstate.Repay {
		sha.Write(common.BigToHash(big.NewInt(int64(l.AccountNonce))).Bytes())
		sha.Write([]byte(l.Status))
		sha.Write(l.RelayerAddress.Bytes())
		sha.Write(l.UserAddress.Bytes())
		sha.Write(l.LendingToken.Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.Term))).Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.LendingTradeId))).Bytes())
	} else if l.Type == lendingstate.TopUp {
		sha.Write(common.BigToHash(big.NewInt(int64(l.AccountNonce))).Bytes())
		sha.Write([]byte(l.Status))
		sha.Write(l.RelayerAddress.Bytes())
		sha.Write(l.UserAddress.Bytes())
		sha.Write(l.LendingToken.Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.Term))).Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.LendingTradeId))).Bytes())
		sha.Write(common.BigToHash(l.Quantity).Bytes())
	} else {
		if l.Status == lendingstate.LendingStatusCancelled {
			sha := sha3.NewLegacyKeccak256()
			sha.Write(l.Hash.Bytes())
			sha.Write(common.BigToHash(big.NewInt(int64(l.AccountNonce))).Bytes())
			sha.Write(l.UserAddress.Bytes())
			sha.Write(common.BigToHash(big.NewInt(int64(l.LendingId))).Bytes())
			sha.Write([]byte(l.Status))
			sha.Write(l.RelayerAddress.Bytes())
		} else if l.Status == lendingstate.LendingStatusNew {
			sha.Write(l.RelayerAddress.Bytes())
			sha.Write(l.UserAddress.Bytes())
			if borrowing {
				sha.Write(l.CollateralToken.Bytes())
			}
			sha.Write(l.LendingToken.Bytes())
			sha.Write(common.BigToHash(l.Quantity).Bytes())
			sha.Write(common.BigToHash(big.NewInt(int64(l.Term))).Bytes())
			if l.Type == lendingstate.Limit {
				sha.Write(common.BigToHash(big.NewInt(int64(l.Interest))).Bytes())
			}
			sha.Write([]byte(l.Side))
			sha.Write([]byte(l.Status))
			sha.Write([]byte(l.Type))
			sha.Write(common.BigToHash(big.NewInt(int64(l.AccountNonce))).Bytes())
			sha.Write(common.BigToHash(big.NewInt(int64(l.LendingTradeId))).Bytes())
			if borrowing {
				autoTopUp := int64(0)
				if l.AutoTopUp {
					autoTopUp = int64(1)
				}
				sha.Write(common.BigToHash(big.NewInt(autoTopUp)).Bytes())
			}
		}
	}

	return common.BytesToHash(sha.Sum(nil))
}
