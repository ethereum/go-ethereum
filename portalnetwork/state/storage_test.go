package state

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/stretchr/testify/require"
)

func TestStorage(t *testing.T) {
	storage := storage.NewMockStorage()
	stateStorage := NewStateStorage(storage)
	testfiles := []string{"account_trie_node.yaml", "contract_storage_trie_node.yaml", "contract_bytecode.yaml"}
	for _, file := range testfiles {
		cases, err := getTestCases(file)
		require.NoError(t, err)
		for _, tt := range cases {
			contentKey := hexutil.MustDecode(tt.ContentKey)
			contentId := defaultContentIdFunc(contentKey)
			err = stateStorage.Put(contentKey, contentId, hexutil.MustDecode(tt.ContentValueOffer))
			require.NoError(t, err)
			res, err := stateStorage.Get(contentKey, contentId)
			require.NoError(t, err)
			require.Equal(t, hexutil.MustDecode(tt.ContentValueRetrieval), res)
		}
	}
}
