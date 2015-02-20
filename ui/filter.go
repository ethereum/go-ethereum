package ui

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethutil"
)

func fromHex(s string) []byte {
	if len(s) > 1 {
		if s[0:2] == "0x" {
			s = s[2:]
		}
		return ethutil.Hex2Bytes(s)
	}
	return nil
}

func NewFilterFromMap(object map[string]interface{}, eth core.Backend) *core.Filter {
	filter := core.NewFilter(eth)

	if object["earliest"] != nil {
		val := ethutil.NewValue(object["earliest"])
		filter.SetEarliestBlock(val.Int())
	}

	if object["latest"] != nil {
		val := ethutil.NewValue(object["latest"])
		filter.SetLatestBlock(val.Int())
	}

	if object["address"] != nil {
		//val := ethutil.NewValue(object["address"])
		//filter.SetAddress(fromHex(val.Str()))
	}

	if object["max"] != nil {
		val := ethutil.NewValue(object["max"])
		filter.SetMax(int(val.Uint()))
	}

	if object["skip"] != nil {
		val := ethutil.NewValue(object["skip"])
		filter.SetSkip(int(val.Uint()))
	}

	if object["topics"] != nil {
		filter.SetTopics(MakeTopics(object["topics"]))
	}

	return filter
}

// Conversion methodn
func mapToAccountChange(m map[string]interface{}) (d core.AccountChange) {
	if str, ok := m["id"].(string); ok {
		d.Address = fromHex(str)
	}

	if str, ok := m["at"].(string); ok {
		d.StateAddress = fromHex(str)
	}

	return
}

// data can come in in the following formats:
// ["aabbccdd", {id: "ccddee", at: "11223344"}], "aabbcc", {id: "ccddee", at: "1122"}
func MakeTopics(v interface{}) (d [][]byte) {
	if str, ok := v.(string); ok {
		d = append(d, fromHex(str))
	} else if slice, ok := v.([]string); ok {
		for _, item := range slice {
			d = append(d, fromHex(item))
		}
	}
	return
}
