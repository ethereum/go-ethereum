package eth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
)

type Web3Service struct {
	stack *node.Node
}

func NewWeb3Service(stack *node.Node) *Web3Service {
	return &Web3Service{stack}
}

func (s *Web3Service) ClientVersion() string {
	return s.stack.Server().Name
}

func (s *Web3Service) Sha3(data string) string {
	return common.ToHex(crypto.Sha3(common.FromHex(data)))
}
