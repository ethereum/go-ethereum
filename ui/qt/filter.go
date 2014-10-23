package qt

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethchain"
	"github.com/ethereum/go-ethereum/ui"
	"gopkg.in/qml.v1"
)

func NewFilterFromMap(object map[string]interface{}, eth ethchain.EthManager) *ethchain.Filter {
	filter := ui.NewFilterFromMap(object, eth)

	if object["altered"] != nil {
		filter.Altered = makeAltered(object["altered"])
	}

	return filter
}

func makeAltered(v interface{}) (d []ethchain.AccountChange) {
	if qList, ok := v.(*qml.List); ok {
		var s []interface{}
		qList.Convert(&s)

		fmt.Println(s)

		d = makeAltered(s)
	} else if qMap, ok := v.(*qml.Map); ok {
		var m map[string]interface{}
		qMap.Convert(&m)
		fmt.Println(m)

		d = makeAltered(m)
	}

	return
}
