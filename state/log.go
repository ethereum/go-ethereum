package state

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/ethutil"
)

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

func (self Log) String() string {
	return fmt.Sprintf(`log: %x %x %x`, self.Address, self.Topics, self.Data)
}

type Logs []Log

func (self Logs) RlpData() interface{} {
	data := make([]interface{}, len(self))
	for i, log := range self {
		data[i] = log.RlpData()
	}

	return data
}

func (self Logs) String() string {
	var logs []string
	for _, log := range self {
		logs = append(logs, log.String())
	}
	return "[ " + strings.Join(logs, ", ") + " ]"
}
