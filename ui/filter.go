package ui

import (
	"github.com/ethereum/go-ethereum/chain"
	"github.com/ethereum/go-ethereum/ethutil"
)

func NewFilterFromMap(object map[string]interface{}, eth chain.EthManager) *chain.Filter {
	filter := chain.NewFilter(eth)

	if object["earliest"] != nil {
		val := ethutil.NewValue(object["earliest"])
		filter.SetEarliestBlock(val.Int())
	}

	if object["latest"] != nil {
		val := ethutil.NewValue(object["latest"])
		filter.SetLatestBlock(val.Int())
	}

	if object["to"] != nil {
		val := ethutil.NewValue(object["to"])
		filter.AddTo(ethutil.Hex2Bytes(val.Str()))
	}

	if object["from"] != nil {
		val := ethutil.NewValue(object["from"])
		filter.AddFrom(ethutil.Hex2Bytes(val.Str()))
	}

	if object["max"] != nil {
		val := ethutil.NewValue(object["max"])
		filter.SetMax(int(val.Uint()))
	}

	if object["skip"] != nil {
		val := ethutil.NewValue(object["skip"])
		filter.SetSkip(int(val.Uint()))
	}

	if object["altered"] != nil {
		filter.Altered = makeAltered(object["altered"])
	}

	return filter
}

// Conversion methodn
func mapToAccountChange(m map[string]interface{}) (d chain.AccountChange) {
	if str, ok := m["id"].(string); ok {
		d.Address = ethutil.Hex2Bytes(str)
	}

	if str, ok := m["at"].(string); ok {
		d.StateAddress = ethutil.Hex2Bytes(str)
	}

	return
}

// data can come in in the following formats:
// ["aabbccdd", {id: "ccddee", at: "11223344"}], "aabbcc", {id: "ccddee", at: "1122"}
func makeAltered(v interface{}) (d []chain.AccountChange) {
	if str, ok := v.(string); ok {
		d = append(d, chain.AccountChange{ethutil.Hex2Bytes(str), nil})
	} else if obj, ok := v.(map[string]interface{}); ok {
		d = append(d, mapToAccountChange(obj))
	} else if slice, ok := v.([]interface{}); ok {
		for _, item := range slice {
			d = append(d, makeAltered(item)...)
		}
	}

	return
}
