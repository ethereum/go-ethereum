package ethapi

import (
	"context"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/core/types"
)

func (s *PublicBlockChainAPI) GetRootHash(ctx context.Context, starBlockNr uint64, endBlockNr uint64) (string, error) {
	root, err := s.b.GetRootHash(ctx, starBlockNr, endBlockNr)
	if err != nil {
		return "", err
	}
	return root, nil
}

func (s *PublicBlockChainAPI) GetBorBlockReceipt(ctx context.Context, hash common.Hash) (*types.BorReceipt, error) {
	return s.b.GetBorBlockReceipt(ctx, hash)
}
