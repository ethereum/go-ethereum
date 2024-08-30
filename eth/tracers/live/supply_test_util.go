package live

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/tests"
)

type BlockTest struct {
	bt       *tests.BlockTest
	Expected []supplyInfo `json:"expected"`
}

func btFromChain(db ethdb.Database, chain *core.BlockChain) (*BlockTest, error) {
	bt, err := tests.FromChain(db, chain)
	if err != nil {
		return nil, err
	}
	return &BlockTest{bt: &bt}, nil
}

func (bt *BlockTest) UnmarshalJSON(data []byte) error {
	tmp := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	if err := json.Unmarshal(tmp["expected"], &bt.Expected); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &bt.bt); err != nil {
		return err
	}
	return nil
}

func (bt *BlockTest) MarshalJSON() ([]byte, error) {
	enc, err := json.Marshal(bt.bt)
	if err != nil {
		return nil, err
	}
	// Insert the expected supply info
	result := make(map[string]any)
	if err := json.Unmarshal(enc, &result); err != nil {
		return nil, err
	}
	result["expected"] = bt.Expected
	return json.Marshal(result)
}
