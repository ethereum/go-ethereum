package eth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Web3Service struct {
	e *Ethereum
}

func NewWeb3Service(e *Ethereum) *Web3Service {
	return &Web3Service{e}
}

func (s *Web3Service) ClientVersion() string {
	return s.e.ClientVersion()
}

func (s *Web3Service) Sha3(data string) string {
	return common.ToHex(crypto.Sha3(common.FromHex(data)))
}