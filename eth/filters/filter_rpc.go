package filters

import (
	"sync"
	"time"

	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/core/types"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
	"encoding/json"
	"fmt"
)

var (
	filterTickerTime = 5 * time.Minute
)

// byte will be inferred
const (
	unknownFilterTy = iota
	blockFilterTy
	transactionFilterTy
	logFilterTy
)


type FilterService struct {
	mux              *event.TypeMux

	quit             chan struct{}
	chainDb          ethdb.Database

	filterManager    *FilterSystem

	filterMapMu      sync.RWMutex
	filterMapping    map[string]int // maps between filter internal filter identifiers and external filter identifiers

	logMu            sync.RWMutex
	logQueue         map[int]*logQueue

	blockMu          sync.RWMutex
	blockQueue       map[int]*hashQueue

	transactionMu    sync.RWMutex
	transactionQueue map[int]*hashQueue

									//	messagesMu       sync.RWMutex
									//	messages         map[int]*whisperFilter

	transactMu       sync.Mutex
}

func NewFilterService(chainDb ethdb.Database, mux *event.TypeMux) *FilterService {
	svc := &FilterService{
		mux: mux,
		chainDb: chainDb,
		filterManager: NewFilterSystem(mux),
		filterMapping: make(map[string]int),
		logQueue: make(map[int]*logQueue),
		blockQueue: make(map[int]*hashQueue),
		transactionQueue: make(map[int]*hashQueue),
	}
	go svc.start()
	return svc
}

func (s *FilterService) Stop() {
	close(s.quit)
}

func (s *FilterService) start() {
	timer := time.NewTicker(2 * time.Second)
	defer timer.Stop()
	done:
	for {
		select {
		case <-timer.C:
			s.logMu.Lock()
			for id, filter := range s.logQueue {
				if time.Since(filter.timeout) > filterTickerTime {
					s.filterManager.Remove(id)
					delete(s.logQueue, id)
				}
			}
			s.logMu.Unlock()

			s.blockMu.Lock()
			for id, filter := range s.blockQueue {
				if time.Since(filter.timeout) > filterTickerTime {
					s.filterManager.Remove(id)
					delete(s.blockQueue, id)
				}
			}
			s.blockMu.Unlock()

			s.transactionMu.Lock()
			for id, filter := range s.transactionQueue {
				if time.Since(filter.timeout) > filterTickerTime {
					s.filterManager.Remove(id)
					delete(s.transactionQueue, id)
				}
			}
			s.transactionMu.Unlock()

		//			s.messagesMu.Lock()
		//			for id, filter := range s.messages {
		//				if time.Since(filter.activity()) > filterTickerTime {
		//					s.Whisper().Unwatch(id)
		//					delete(s.messages, id)
		//				}
		//			}
		//			s.messagesMu.Unlock()
		case <-s.quit:
			break done
		}
	}

}

func (s *FilterService) NewBlockFilter() (string, error) {
	externalId, err := newFilterId()
	if err != nil {
		return "", err
	}

	s.blockMu.Lock()
	filter := New(s.chainDb)
	id := s.filterManager.Add(filter)
	s.blockQueue[id] = &hashQueue{timeout: time.Now()}

	filter.BlockCallback = func(block *types.Block, logs vm.Logs) {
		s.blockMu.Lock()
		defer s.blockMu.Unlock()

		if queue := s.blockQueue[id]; queue != nil {
			queue.add(block.Hash())
		}
	}

	defer s.blockMu.Unlock()

	s.filterMapMu.Lock()
	s.filterMapping[externalId] = id
	s.filterMapMu.Unlock()

	return externalId, nil
}

func (s *FilterService) NewPendingTransactionFilter() (string, error) {
	externalId, err := newFilterId()
	if err != nil {
		return "", err
	}

	s.transactionMu.Lock()
	defer s.transactionMu.Unlock()

	filter := New(s.chainDb)
	id := s.filterManager.Add(filter)
	s.transactionQueue[id] = &hashQueue{timeout: time.Now()}

	filter.TransactionCallback = func(tx *types.Transaction) {
		s.transactionMu.Lock()
		defer s.transactionMu.Unlock()

		if queue := s.transactionQueue[id]; queue != nil {
			queue.add(tx.Hash())
		}
	}

	s.filterMapMu.Lock()
	s.filterMapping[externalId] = id
	s.filterMapMu.Unlock()

	return externalId, nil
}

func (s *FilterService) newLogFilter(earliest, latest int64, addresses []common.Address, topics [][]common.Hash) int {
	s.logMu.Lock()
	defer s.logMu.Unlock()

	filter := New(s.chainDb)
	id := s.filterManager.Add(filter)
	s.logQueue[id] = &logQueue{timeout: time.Now()}

	filter.SetBeginBlock(earliest)
	filter.SetEndBlock(latest)
	filter.SetAddresses(addresses)
	filter.SetTopics(topics)
	filter.LogsCallback = func(logs vm.Logs) {
		s.logMu.Lock()
		defer s.logMu.Unlock()

		if queue := s.logQueue[id]; queue != nil {
			queue.add(logs...)
		}
	}

	return id
}

type NewFilterArgs struct {
	FromBlock rpc.BlockNumber
	ToBlock   rpc.BlockNumber
	Addresses []common.Address
	Topics    [][]common.Hash
}

func (args *NewFilterArgs) UnmarshalJSON(data []byte) error {
	type input struct {
		From      *rpc.BlockNumber `json:"fromBlock"`
		ToBlock   *rpc.BlockNumber `json:"toBlock"`
		Addresses interface{}     `json:"address"`
		Topics    interface{}     `json:"topics"`
	}

	var raw input
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.From == nil {
		args.FromBlock = rpc.LatestBlockNumber
	} else {
		args.FromBlock = *raw.From
	}

	if raw.ToBlock == nil {
		args.ToBlock = rpc.LatestBlockNumber
	} else {
		args.ToBlock = *raw.ToBlock
	}

	args.Addresses = []common.Address{}

	if raw.Addresses != nil {
		// raw.Address can contain a single address or an array of addresses
		var addresses []common.Address

		if strAddrs, ok := raw.Addresses.([]interface{}); ok {
			for i, addr := range strAddrs {
				if strAddr, ok := addr.(string); ok {
					if len(strAddr) >= 2 && strAddr[0] == '0' && (strAddr[1] == 'x' || strAddr[1] == 'X') {
						strAddr = strAddr[2:]
					}
					if decAddr, err := hex.DecodeString(strAddr); err == nil {
						addresses = append(addresses, common.BytesToAddress(decAddr))
					} else {
						fmt.Errorf("invalid address given")
					}
				} else {
					return fmt.Errorf("invalid address on index %d", i)
				}
			}
		} else if singleAddr, ok := raw.Addresses.(string); ok {
			if len(singleAddr) >= 2 && singleAddr[0] == '0' && (singleAddr[1] == 'x' || singleAddr[1] == 'X') {
				singleAddr = singleAddr[2:]
			}
			if decAddr, err := hex.DecodeString(singleAddr); err == nil {
				addresses = append(addresses, common.BytesToAddress(decAddr))
			} else {
				fmt.Errorf("invalid address given")
			}
		} else {
			errors.New("invalid address(es) given")
		}
		args.Addresses = addresses
	}

	topicConverter := func(raw string) (common.Hash, error) {
		if len(raw) == 0 {
			return common.Hash{}, nil
		}

		if len(raw) >= 2 && raw[0] == '0' && (raw[1] == 'x' || raw[1] == 'X') {
			raw = raw[2:]
		}

		if decAddr, err := hex.DecodeString(raw); err == nil {
			return common.BytesToHash(decAddr), nil
		}

		return common.Hash{}, errors.New("invalid topic given")
	}

	// topics is an array consisting of strings or arrays of strings
	if raw.Topics != nil {
		topics, ok := raw.Topics.([]interface{})
		if ok {
			parsedTopics := make([][]common.Hash, len(topics))
			for i, topic := range topics {
				if topic == nil {
					parsedTopics[i] = []common.Hash{common.StringToHash("")}
				} else if strTopic, ok := topic.(string); ok {
					if t, err := topicConverter(strTopic); err != nil {
						return fmt.Errorf("invalid topic on index %d", i)
					} else {
						parsedTopics[i] = []common.Hash{t}
					}
				} else if arrTopic, ok := topic.([]interface{}); ok {
					parsedTopics[i] = make([]common.Hash, len(arrTopic))
					for j := 0; j < len(parsedTopics[i]); i++ {
						if arrTopic[j] == nil {
							parsedTopics[i][j] = common.StringToHash("")
						} else if str, ok := arrTopic[j].(string); ok {
							if t, err := topicConverter(str); err != nil {
								return fmt.Errorf("invalid topic on index %d", i)
							} else {
								parsedTopics[i] = []common.Hash{t}
							}
						} else {
							fmt.Errorf("topic[%d][%d] not a string", i, j)
						}
					}
				} else {
					return fmt.Errorf("topic[%d] invalid", i)
				}
			}
			args.Topics = parsedTopics
		}
	}

	return nil
}

func (s *FilterService) NewFilter(args NewFilterArgs) (string, error) {
	externalId, err := newFilterId()
	if err != nil {
		return "", err
	}

	var id int
	if len(args.Addresses) > 0 {
		id = s.newLogFilter(args.FromBlock.Int64(), args.ToBlock.Int64(), args.Addresses, args.Topics)
	} else {
		id = s.newLogFilter(args.FromBlock.Int64(), args.ToBlock.Int64(), nil, args.Topics)
	}

	s.filterMapMu.Lock()
	s.filterMapping[externalId] = id
	s.filterMapMu.Unlock()

	return externalId, nil
}

func (s *FilterService) GetLogs(args NewFilterArgs) (vm.Logs) {
	filter := New(s.chainDb)
	filter.SetBeginBlock(args.FromBlock.Int64())
	filter.SetEndBlock(args.ToBlock.Int64())
	filter.SetAddresses(args.Addresses)
	filter.SetTopics(args.Topics)

	return filter.Find()
}

func (s *FilterService) UninstallFilter(filterId string) bool {
	s.filterMapMu.Lock()
	defer s.filterMapMu.Unlock()

	id, ok := s.filterMapping[filterId]
	if !ok {
		return false
	}

	defer s.filterManager.Remove(id)
	delete(s.filterMapping, filterId)

	if _, ok := s.logQueue[id]; ok {
		s.logMu.Lock()
		defer s.logMu.Unlock()
		delete(s.logQueue, id)
		return true
	}
	if _, ok := s.blockQueue[id]; ok {
		s.blockMu.Lock()
		defer s.blockMu.Unlock()
		delete(s.blockQueue, id)
		return true
	}
	if _, ok := s.transactionQueue[id]; ok {
		s.transactionMu.Lock()
		defer s.transactionMu.Unlock()
		delete(s.transactionQueue, id)
		return true
	}

	return false
}

func (s *FilterService) getFilterType(id int) byte {
	if _, ok := s.blockQueue[id]; ok {
		return blockFilterTy
	} else if _, ok := s.transactionQueue[id]; ok {
		return transactionFilterTy
	} else if _, ok := s.logQueue[id]; ok {
		return logFilterTy
	}

	return unknownFilterTy
}

func (s *FilterService) blockFilterChanged(id int) []common.Hash {
	s.blockMu.Lock()
	defer s.blockMu.Unlock()

	if s.blockQueue[id] != nil {
		return s.blockQueue[id].get()
	}
	return []common.Hash{}
}

func (s *FilterService) transactionFilterChanged(id int) []common.Hash {
	s.blockMu.Lock()
	defer s.blockMu.Unlock()

	if s.transactionQueue[id] != nil {
		return s.transactionQueue[id].get()
	}
	return []common.Hash{}
}

func (s *FilterService) logFilterChanged(id int) vm.Logs {
	s.logMu.Lock()
	defer s.logMu.Unlock()

	if s.logQueue[id] != nil {
		return s.logQueue[id].get()
	}
	return vm.Logs{}
}

func (s *FilterService) GetFilterLogs(filterId string) vm.Logs {
	id, ok := s.filterMapping[filterId]
	if !ok {
		return vm.Logs{}
	}

	if filter := s.filterManager.Get(id); filter != nil {
		return filter.Find()
	}

	return vm.Logs{}
}

func (s *FilterService) GetFilterChanges(filterId string) interface{} {
	s.filterMapMu.Lock()
	id, ok := s.filterMapping[filterId]
	s.filterMapMu.Unlock()

	if !ok { // filter not found
		return nil
	}

	switch s.getFilterType(id) {
	case blockFilterTy:
		return s.blockFilterChanged(id)
	case transactionFilterTy:
		return s.transactionFilterChanged(id)
	case logFilterTy:
		return s.logFilterChanged(id)
	}

	return nil
}

type logQueue struct {
	mu      sync.Mutex

	logs    vm.Logs
	timeout time.Time
	id      int
}

func (l *logQueue) add(logs ...*vm.Log) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logs = append(l.logs, logs...)
}

func (l *logQueue) get() vm.Logs {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.timeout = time.Now()
	tmp := l.logs
	l.logs = nil
	return tmp
}

type hashQueue struct {
	mu      sync.Mutex

	hashes  []common.Hash
	timeout time.Time
	id      int
}

func (l *hashQueue) add(hashes ...common.Hash) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.hashes = append(l.hashes, hashes...)
}

func (l *hashQueue) get() []common.Hash {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.timeout = time.Now()
	tmp := l.hashes
	l.hashes = nil
	return tmp
}

func newFilterId() (string, error) {
	var subid [16]byte
	n, _ := rand.Read(subid[:])
	if n != 16 {
		return "", errors.New("Unable to generate filter id")
	}
	return "0x" + hex.EncodeToString(subid[:]), nil
}
