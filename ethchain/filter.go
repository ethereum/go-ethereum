package ethchain

import (
	"bytes"
	"fmt"
	"math"

	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
	"gopkg.in/qml.v1"
)

type data struct {
	id, address []byte
}

// Filtering interface
type Filter struct {
	eth      EthManager
	earliest int64
	latest   int64
	skip     int
	from, to [][]byte
	max      int

	altered []data

	BlockCallback   func(*Block)
	MessageCallback func(ethstate.Messages)
}

// Create a new filter which uses a bloom filter on blocks to figure out whether a particular block
// is interesting or not.
func NewFilter(eth EthManager) *Filter {
	return &Filter{eth: eth}
}

func NewFilterFromMap(object map[string]interface{}, eth EthManager) *Filter {
	filter := NewFilter(eth)

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
		filter.altered = makeAltered(object["altered"])
	}

	return filter
}

func (self *Filter) AddAltered(id, address []byte) {
	self.altered = append(self.altered, data{id, address})
}

// Set the earliest and latest block for filtering.
// -1 = latest block (i.e., the current block)
// hash = particular hash from-to
func (self *Filter) SetEarliestBlock(earliest int64) {
	self.earliest = earliest
}

func (self *Filter) SetLatestBlock(latest int64) {
	self.latest = latest
}

func (self *Filter) SetFrom(addr [][]byte) {
	self.from = addr
}

func (self *Filter) AddFrom(addr []byte) {
	self.from = append(self.from, addr)
}

func (self *Filter) SetTo(addr [][]byte) {
	self.to = addr
}

func (self *Filter) AddTo(addr []byte) {
	self.to = append(self.to, addr)
}

func (self *Filter) SetMax(max int) {
	self.max = max
}

func (self *Filter) SetSkip(skip int) {
	self.skip = skip
}

// Run filters messages with the current parameters set
func (self *Filter) Find() []*ethstate.Message {
	var earliestBlockNo uint64 = uint64(self.earliest)
	if self.earliest == -1 {
		earliestBlockNo = self.eth.BlockChain().CurrentBlock.Number.Uint64()
	}
	var latestBlockNo uint64 = uint64(self.latest)
	if self.latest == -1 {
		latestBlockNo = self.eth.BlockChain().CurrentBlock.Number.Uint64()
	}

	var (
		messages []*ethstate.Message
		block    = self.eth.BlockChain().GetBlockByNumber(latestBlockNo)
		quit     bool
	)
	for i := 0; !quit && block != nil; i++ {
		// Quit on latest
		switch {
		case block.Number.Uint64() == earliestBlockNo:
			quit = true
		case self.max <= len(messages):
			break
		}

		// Use bloom filtering to see if this block is interesting given the
		// current parameters
		if self.bloomFilter(block) {
			// Get the messages of the block
			msgs, err := self.eth.StateManager().GetMessages(block)
			if err != nil {
				chainlogger.Warnln("err: filter get messages ", err)

				break
			}

			messages = append(messages, self.FilterMessages(msgs)...)
		}

		block = self.eth.BlockChain().GetBlock(block.PrevHash)
	}

	skip := int(math.Min(float64(len(messages)), float64(self.skip)))

	return messages[skip:]
}

func includes(addresses [][]byte, a []byte) (found bool) {
	for _, addr := range addresses {
		if bytes.Compare(addr, a) == 0 {
			return true
		}
	}

	return
}

func (self *Filter) FilterMessages(msgs []*ethstate.Message) []*ethstate.Message {
	var messages []*ethstate.Message

	// Filter the messages for interesting stuff
	for _, message := range msgs {
		if len(self.to) > 0 && !includes(self.to, message.To) {
			continue
		}

		if len(self.from) > 0 && !includes(self.from, message.From) {
			continue
		}

		var match bool
		if len(self.altered) == 0 {
			match = true
		}

		for _, item := range self.altered {
			if len(item.id) > 0 && bytes.Compare(message.To, item.id) != 0 {
				continue
			}

			if len(item.address) > 0 && !includes(message.ChangedAddresses, item.address) {
				continue
			}

			match = true
			break
		}

		if !match {
			continue
		}

		messages = append(messages, message)
	}

	return messages
}

func (self *Filter) bloomFilter(block *Block) bool {
	fk := append([]byte("bloom"), block.Hash()...)
	bin, err := self.eth.Db().Get(fk)
	if err != nil {
		panic(err)
	}

	bloom := NewBloomFilter(bin)

	var fromIncluded, toIncluded bool
	if len(self.from) > 0 {
		for _, from := range self.from {
			if bloom.Search(from) {
				fromIncluded = true
				break
			}
		}
	} else {
		fromIncluded = true
	}

	if len(self.to) > 0 {
		for _, to := range self.to {
			if bloom.Search(to) {
				toIncluded = true
				break
			}
		}
	} else {
		toIncluded = true
	}

	return fromIncluded && toIncluded
}

// Conversion methodn
func mapToData(m map[string]interface{}) (d data) {
	if str, ok := m["id"].(string); ok {
		d.id = ethutil.Hex2Bytes(str)
	}

	if str, ok := m["at"].(string); ok {
		d.address = ethutil.Hex2Bytes(str)
	}

	return
}

// data can come in in the following formats:
// ["aabbccdd", {id: "ccddee", at: "11223344"}], "aabbcc", {id: "ccddee", at: "1122"}
func makeAltered(v interface{}) (d []data) {
	if str, ok := v.(string); ok {
		d = append(d, data{ethutil.Hex2Bytes(str), nil})
	} else if obj, ok := v.(map[string]interface{}); ok {
		d = append(d, mapToData(obj))
	} else if slice, ok := v.([]interface{}); ok {
		for _, item := range slice {
			d = append(d, makeAltered(item)...)
		}
	} else if qList, ok := v.(*qml.List); ok {
		var s []interface{}
		qList.Convert(&s)

		fmt.Println(s)

		d = makeAltered(s)
	} else if qMap, ok := v.(*qml.Map); ok {
		var m map[string]interface{}
		qMap.Convert(&m)
		fmt.Println(m)

		d = makeAltered(m)
	} else {
		panic(fmt.Sprintf("makeAltered err (unknown conversion): %T\n", v))
	}

	return
}
