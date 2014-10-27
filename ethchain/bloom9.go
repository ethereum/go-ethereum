package ethchain

import "github.com/ethereum/go-ethereum/vm"

func CreateBloom(txs Transactions) uint64 {
	var bin uint64
	for _, tx := range txs {
		bin |= logsBloom(tx.logs)
	}

	return bin
}

func logsBloom(logs []vm.Log) uint64 {
	var bin uint64
	for _, log := range logs {
		data := []byte{log.Address}
		for _, topic := range log.Topics {
			data = append(data, topic.Bytes())
		}
		data = append(data, log.Data)

		for _, b := range data {
			bin |= bloom9(b)
		}
	}

	return bin
}

func bloom9(b []byte) uint64 {
	var r uint64
	for _, i := range []int{0, 2, 4} {
		r |= 1 << (b[i+1] + 256*(b[i]&1))
	}

	return r
}
