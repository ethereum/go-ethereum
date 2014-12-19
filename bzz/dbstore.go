// disk storage layer for the package blockhash
// inefficient work-in-progress version

package blockhash

import (
	//	"crypto/sha256"
	//	"encoding/binary"
	"bytes"
	"fmt"
	// "github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"path"
)

type dpaDBStorage struct {
	dpaStorage
	//	db *ethdb.LDBDatabase
	db *LDBDatabase
}

func (s *dpaDBStorage) add(entry *dpaStoreReq) {

	data := ethutil.Encode([]interface{}{entry.data, entry.size})

	s.db.Put(entry.hash, data)

}

func (s *dpaDBStorage) find(hash HashType) (entry dpaNode) {

	fmt.Printf("mao")

	data, err := s.db.Get(hash)
	if err != nil {
		panic("hi")
	}
	fmt.Printf("mao")

	dec := rlp.NewListStream(bytes.NewReader(data), uint64(len(data)))
	dec.Decode(&entry)

	return

}

func (s *dpaDBStorage) process_store(req *dpaStoreReq) {

	s.add(req)

	if s.chain != nil {
		s.chain.store_chn <- req
	}

}

func (s *dpaDBStorage) process_retrieve(req *dpaRetrieveReq) {

	fmt.Printf("mao")
	entry := s.find(req.hash)

	fmt.Printf("%v", entry.size)
	if entry.data == nil {
		if s.chain != nil {
			s.chain.retrieve_chn <- req
			return
		}
	}

	res := new(dpaRetrieveRes)
	if entry.data != nil {
		res.dpaNode = entry
	}
	res.req_id = req.req_id
	req.result_chn <- res

}

func (s *dpaDBStorage) Init(ch *dpaStorage) {

	// dbPath := path.Join(ethutil.Config.ExecPath, "bzz")
	dbPath := path.Join(".", "bzz")

	// Open the db
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return
	}

	//	s.db = &ethdb.LDBDatabase{db: db, comp: false}
	s.db = &LDBDatabase{db: db, comp: false}

}

func (s *dpaDBStorage) Run() {

	for {
		bb := true
		for bb {
			select {
			case store := <-s.store_chn:
				s.process_store(store)
			default:
				bb = false
			}
		}
		select {
		case store := <-s.store_chn:
			s.process_store(store)
		case retrv := <-s.retrieve_chn:
			s.process_retrieve(retrv)
		}
	}

}
