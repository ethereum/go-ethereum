package ethapi

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// DbGet returns the raw value of a key stored in the database.
func (api *PrivateDebugAPI) DbGet(key string) (hexutil.Bytes, error) {
	blob, err := common.ParseHexOrString(key)
	if err != nil {
		return nil, err
	}
	return api.b.ChainDb().Get(blob)
}

// DbAncient retrieves an ancient binary blob from the append-only immutable files.
// It is a mapping to the `AncientReaderOp.Ancient` method
func (api *PrivateDebugAPI) DbAncient(kind string, number uint64) (hexutil.Bytes, error) {
	return api.b.ChainDb().Ancient(kind, number)
}

// DbAncients returns the ancient item numbers in the ancient store.
// It is a mapping to the `AncientReaderOp.Ancients` method
func (api *PrivateDebugAPI) DbAncients() (uint64, error) {
	return api.b.ChainDb().Ancients()
}
