package state

import "github.com/ethereum/go-ethereum/ethutil"

type Log struct {
	Address []byte
	Topics  [][]byte
	Data    []byte
}

func NewLogFromValue(decoder *ethutil.Value) Log {
	log := Log{
		Address: decoder.Get(0).Bytes(),
		Data:    decoder.Get(2).Bytes(),
	}

	it := decoder.Get(1).NewIterator()
	for it.Next() {
		log.Topics = append(log.Topics, it.Value().Bytes())
	}

	return log
}

func (self Log) RlpData() interface{} {
	return []interface{}{self.Address, ethutil.ByteSliceToInterface(self.Topics), self.Data}
}

type Logs []Log

func (self Logs) RlpData() interface{} {
	data := make([]interface{}, len(self))
	for i, log := range self {
		data[i] = log.RlpData()
	}

	return data
}
