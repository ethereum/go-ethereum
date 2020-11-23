package ethapi

import (
	"context"
)

// GetRootHash returns root hash for given start and end block
func (s *PublicBlockChainAPI) GetRootHash(ctx context.Context, starBlockNr uint64, endBlockNr uint64) (string, error) {
	root, err := s.b.GetRootHash(ctx, starBlockNr, endBlockNr)
	if err != nil {
		return "", err
	}
	return root, nil
}
