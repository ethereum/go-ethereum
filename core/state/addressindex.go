package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	"sync"
)

var emptyPointer = "EMPTY"

type AddressIndex struct {
	addresses map[common.Address]bool
	db        *leveldb.Database
	lock      sync.Mutex
	readonly  bool
}

const MaxAddressesToCommit = 1000

var GlobalAddressIndex *AddressIndex

func CreateAddressIndex(file string) error {
	db, err := leveldb.New(file, 2000, 1024, "eth/addressindex", false)
	if err != nil {
		return err
	}
	GlobalAddressIndex = &AddressIndex{
		db:        db,
		addresses: make(map[common.Address]bool),
		readonly:  false,
	}
	return nil
}

var emptyValue []byte

func isZeroAddress(addr common.Address) bool {
	return addr[0] == 0 && addr[1] == 0 && addr[2] == 0 && addr[3] == 0 && addr[4] == 0
}

func (s *AddressIndex) SetReadOnly(ro bool) {
	s.readonly = ro
}

func (s *AddressIndex) AddressSeen(addr common.Address) {
	if isZeroAddress(addr) || s.readonly {
		// these are special accounts
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.addresses[addr] = true
	if len(s.addresses) > MaxAddressesToCommit {
		s.flush()
	}
}

func (s *AddressIndex) flush() {
	batch := s.db.NewBatch()
	for addr, _ := range s.addresses {
		batch.Put(addr.Bytes(), emptyValue)
	}
	batch.Write()
	s.addresses = make(map[common.Address]bool)
}

func (s *AddressIndex) IterateSeenAddresses(callback func(common.Address) bool) {
	it := s.db.NewIterator([]byte{}, []byte{})
	defer it.Release()

	for it.Next() {
		if !callback(common.BytesToAddress(it.Key())) {
			return
		}
	}
}

func (s *AddressIndex) HasAddress(addr common.Address) bool {
	has, _ := s.db.Has(addr.Bytes())
	return has
}

func (s *AddressIndex) Stop() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.flush()
}
