package ethchain

import (
	"bytes"
	"fmt"

	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
)

// Filtering interface
type Filter struct {
	eth      EthManager
	earliest []byte
	latest   []byte
	skip     int
	from, to [][]byte
	max      int
}

// Create a new filter which uses a bloom filter on blocks to figure out whether a particular block
// is interesting or not.
func NewFilter(eth EthManager) *Filter {
	return &Filter{eth: eth}
}

func NewFilterFromMap(object map[string]interface{}, eth EthManager) *Filter {
	filter := NewFilter(eth)

	if object["earliest"] != nil {
		earliest := object["earliest"]
		if e, ok := earliest.(string); ok {
			filter.SetEarliestBlock(ethutil.Hex2Bytes(e))
		} else {
			filter.SetEarliestBlock(earliest)
		}
	}

	if object["latest"] != nil {
		latest := object["latest"]
		if l, ok := latest.(string); ok {
			filter.SetLatestBlock(ethutil.Hex2Bytes(l))
		} else {
			filter.SetLatestBlock(latest)
		}
	}

	if object["to"] != nil {
		filter.AddTo(ethutil.Hex2Bytes(object["to"].(string)))
	}

	if object["from"] != nil {
		filter.AddFrom(ethutil.Hex2Bytes(object["from"].(string)))
	}

	if object["max"] != nil {
		filter.SetMax(object["max"].(int))
	}

	if object["skip"] != nil {
		filter.SetSkip(object["skip"].(int))
	}

	return filter
}

// Set the earliest and latest block for filtering.
// -1 = latest block (i.e., the current block)
// hash = particular hash from-to
func (self *Filter) SetEarliestBlock(earliest interface{}) {
	e := ethutil.NewValue(earliest)

	// Check for -1 (latest) otherwise assume bytes
	if e.Int() == -1 {
		self.earliest = self.eth.BlockChain().CurrentBlock.Hash()
	} else if e.Len() > 0 {
		self.earliest = e.Bytes()
	} else {
		panic(fmt.Sprintf("earliest has to be either -1 or a valid hash: %v (%T)", e, e.Val))
	}
}

func (self *Filter) SetLatestBlock(latest interface{}) {
	l := ethutil.NewValue(latest)

	// Check for -1 (latest) otherwise assume bytes
	if l.Int() == -1 {
		self.latest = self.eth.BlockChain().CurrentBlock.Hash()
	} else if l.Len() > 0 {
		self.latest = l.Bytes()
	} else {
		panic(fmt.Sprintf("latest has to be either -1 or a valid hash: %v", l))
	}
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
	self.from = append(self.to, addr)
}

func (self *Filter) SetMax(max int) {
	self.max = max
}

func (self *Filter) SetSkip(skip int) {
	self.skip = skip
}

// Run filters messages with the current parameters set
func (self *Filter) Find() []*ethstate.Message {
	var messages []*ethstate.Message

	block := self.eth.BlockChain().GetBlock(self.latest)

	// skip N blocks (useful for pagination)
	if self.skip > 0 {
		for i := 0; i < i; i++ {
			block = self.eth.BlockChain().GetBlock(block.PrevHash)
		}
	}

	// Start block filtering
	quit := false
	for i := 1; !quit && block != nil; i++ {
		// Mark last check
		if self.max == i || (len(self.earliest) > 0 && bytes.Compare(block.Hash(), self.earliest) == 0) {
			quit = true
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

	return messages
}

func (self *Filter) FilterMessages(msgs []*ethstate.Message) []*ethstate.Message {
	var messages []*ethstate.Message
	includes := func(addresses [][]byte, a []byte) (found bool) {
		for _, addr := range addresses {
			if bytes.Compare(addr, a) == 0 {
				return true
			}
		}

		return
	}
	// Filter the messages for interesting stuff
	for _, message := range msgs {
		if len(self.to) > 0 && !includes(self.to, message.To) {
			continue
		}

		if len(self.from) > 0 && !includes(self.from, message.From) {
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
