package chain

import (
	"bytes"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/chain/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
)

type AccountChange struct {
	Address, StateAddress []byte
}

// Filtering interface
type Filter struct {
	eth      EthManager
	earliest int64
	latest   int64
	skip     int
	from, to [][]byte
	max      int

	Altered []AccountChange

	BlockCallback   func(*types.Block)
	MessageCallback func(state.Messages)
}

// Create a new filter which uses a bloom filter on blocks to figure out whether a particular block
// is interesting or not.
func NewFilter(eth EthManager) *Filter {
	return &Filter{eth: eth}
}

func (self *Filter) AddAltered(address, stateAddress []byte) {
	self.Altered = append(self.Altered, AccountChange{address, stateAddress})
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
func (self *Filter) Find() []*state.Message {
	var earliestBlockNo uint64 = uint64(self.earliest)
	if self.earliest == -1 {
		earliestBlockNo = self.eth.ChainManager().CurrentBlock.Number.Uint64()
	}
	var latestBlockNo uint64 = uint64(self.latest)
	if self.latest == -1 {
		latestBlockNo = self.eth.ChainManager().CurrentBlock.Number.Uint64()
	}

	var (
		messages []*state.Message
		block    = self.eth.ChainManager().GetBlockByNumber(latestBlockNo)
		quit     bool
	)
	for i := 0; !quit && block != nil; i++ {
		// Quit on latest
		switch {
		case block.Number.Uint64() == earliestBlockNo, block.Number.Uint64() == 0:
			quit = true
		case self.max <= len(messages):
			break
		}

		// Use bloom filtering to see if this block is interesting given the
		// current parameters
		if self.bloomFilter(block) {
			// Get the messages of the block
			msgs, err := self.eth.BlockManager().GetMessages(block)
			if err != nil {
				chainlogger.Warnln("err: filter get messages ", err)

				break
			}

			messages = append(messages, self.FilterMessages(msgs)...)
		}

		block = self.eth.ChainManager().GetBlock(block.PrevHash)
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

func (self *Filter) FilterMessages(msgs []*state.Message) []*state.Message {
	var messages []*state.Message

	// Filter the messages for interesting stuff
	for _, message := range msgs {
		if len(self.to) > 0 && !includes(self.to, message.To) {
			continue
		}

		if len(self.from) > 0 && !includes(self.from, message.From) {
			continue
		}

		var match bool
		if len(self.Altered) == 0 {
			match = true
		}

		for _, accountChange := range self.Altered {
			if len(accountChange.Address) > 0 && bytes.Compare(message.To, accountChange.Address) != 0 {
				continue
			}

			if len(accountChange.StateAddress) > 0 && !includes(message.ChangedAddresses, accountChange.StateAddress) {
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

func (self *Filter) bloomFilter(block *types.Block) bool {
	var fromIncluded, toIncluded bool
	if len(self.from) > 0 {
		for _, from := range self.from {
			if types.BloomLookup(block.LogsBloom, from) || bytes.Equal(block.Coinbase, from) {
				fromIncluded = true
				break
			}
		}
	} else {
		fromIncluded = true
	}

	if len(self.to) > 0 {
		for _, to := range self.to {
			if types.BloomLookup(block.LogsBloom, ethutil.U256(new(big.Int).Add(ethutil.Big1, ethutil.BigD(to))).Bytes()) || bytes.Equal(block.Coinbase, to) {
				toIncluded = true
				break
			}
		}
	} else {
		toIncluded = true
	}

	return fromIncluded && toIncluded
}
