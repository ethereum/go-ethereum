// nolint
package rawdb

import (
	"fmt"

	json "github.com/json-iterator/go"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/generics"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

var (
	lastMilestone      = []byte("LastMilestone")
	lockFieldKey       = []byte("LockField")
	futureMilestoneKey = []byte("FutureMilestoneField")
)

type Finality struct {
	Block uint64
	Hash  common.Hash
}

type LockField struct {
	Val    bool
	Block  uint64
	Hash   common.Hash
	IdList map[string]struct{}
}

type FutureMilestoneField struct {
	Order []uint64
	List  map[uint64]common.Hash
}

func (f *Finality) set(block uint64, hash common.Hash) {
	f.Block = block
	f.Hash = hash
}

type Milestone struct {
	Finality
}

func (m *Milestone) clone() *Milestone {
	return &Milestone{}
}

func (m *Milestone) block() (uint64, common.Hash) {
	return m.Block, m.Hash
}

func ReadFinality[T BlockFinality[T]](db ethdb.KeyValueReader) (uint64, common.Hash, error) {
	lastTV, key := getKey[T]()

	data, err := db.Get(key)
	if err != nil {
		return 0, common.Hash{}, fmt.Errorf("%w: empty response for %s", err, string(key))
	}

	if len(data) == 0 {
		return 0, common.Hash{}, fmt.Errorf("%w for %s", ErrEmptyLastFinality, string(key))
	}

	if err = json.Unmarshal(data, lastTV); err != nil {
		log.Error(fmt.Sprintf("Unable to unmarshal the last %s block number in database", string(key)), "err", err)

		return 0, common.Hash{}, fmt.Errorf("%w(%v) for %s, data %v(%q)",
			ErrIncorrectFinality, err, string(key), data, string(data))
	}

	block, hash := lastTV.block()

	return block, hash, nil
}

func WriteLastFinality[T BlockFinality[T]](db ethdb.KeyValueWriter, block uint64, hash common.Hash) error {
	lastTV, key := getKey[T]()

	lastTV.set(block, hash)

	enc, err := json.Marshal(lastTV)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to marshal the %s struct", string(key)), "err", err)

		return fmt.Errorf("%w: %v for %s struct", ErrIncorrectFinalityToStore, err, string(key))
	}

	if err := db.Put(key, enc); err != nil {
		log.Error(fmt.Sprintf("Failed to store the %s struct", string(key)), "err", err)

		return fmt.Errorf("%w: %v for %s struct", ErrDBNotResponding, err, string(key))
	}

	return nil
}

type BlockFinality[T any] interface {
	set(block uint64, hash common.Hash)
	clone() T
	block() (uint64, common.Hash)
}

func getKey[T BlockFinality[T]]() (T, []byte) {
	lastT := generics.Empty[T]().clone()

	var key []byte

	switch any(lastT).(type) {
	case *Milestone:
		key = lastMilestone
	case *Checkpoint:
		key = lastCheckpoint
	}

	return lastT, key
}

func WriteLockField(db ethdb.KeyValueWriter, val bool, block uint64, hash common.Hash, idListMap map[string]struct{}) error {

	lockField := LockField{
		Val:    val,
		Block:  block,
		Hash:   hash,
		IdList: idListMap,
	}

	key := lockFieldKey

	enc, err := json.Marshal(lockField)
	if err != nil {
		log.Error("Failed to marshal the lock field struct", "err", err)

		return fmt.Errorf("%w: %v for lock field struct", ErrIncorrectLockFieldToStore, err)
	}

	if err := db.Put(key, enc); err != nil {
		log.Error("Failed to store the lock field struct", "err", err)

		return fmt.Errorf("%w: %v for lock field struct", ErrDBNotResponding, err)
	}

	return nil
}

func ReadLockField(db ethdb.KeyValueReader) (bool, uint64, common.Hash, map[string]struct{}, error) {
	key := lockFieldKey
	lockField := LockField{}

	data, err := db.Get(key)
	if err != nil {
		return false, 0, common.Hash{}, nil, fmt.Errorf("%w: empty response for lock field", err)
	}

	if len(data) == 0 {
		return false, 0, common.Hash{}, nil, fmt.Errorf("%w for %s", ErrIncorrectLockField, string(key))
	}

	if err = json.Unmarshal(data, &lockField); err != nil {
		log.Error(fmt.Sprintf("Unable to unmarshal the lock field in database"), "err", err)

		return false, 0, common.Hash{}, nil, fmt.Errorf("%w(%v) for lock field , data %v(%q)",
			ErrIncorrectLockField, err, data, string(data))
	}

	val, block, hash, idList := lockField.Val, lockField.Block, lockField.Hash, lockField.IdList

	return val, block, hash, idList, nil
}

func WriteFutureMilestoneList(db ethdb.KeyValueWriter, order []uint64, list map[uint64]common.Hash) error {

	futureMilestoneField := FutureMilestoneField{
		Order: order,
		List:  list,
	}

	key := futureMilestoneKey

	enc, err := json.Marshal(futureMilestoneField)
	if err != nil {
		log.Error("Failed to marshal the future milestone field struct", "err", err)

		return fmt.Errorf("%w: %v for future milestone field struct", ErrIncorrectFutureMilestoneFieldToStore, err)
	}

	if err = db.Put(key, enc); err != nil {
		log.Error("Failed to store the future milestone field struct", "err", err)

		return fmt.Errorf("%w: %v for future milestone field struct", ErrDBNotResponding, err)
	}

	return nil
}

func ReadFutureMilestoneList(db ethdb.KeyValueReader) ([]uint64, map[uint64]common.Hash, error) {
	key := futureMilestoneKey
	futureMilestoneField := FutureMilestoneField{}

	data, err := db.Get(key)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: empty response for future milestone field", err)
	}

	if len(data) == 0 {
		return nil, nil, fmt.Errorf("%w for %s", ErrIncorrectLockField, string(key))
	}

	if err = json.Unmarshal(data, &futureMilestoneField); err != nil {
		log.Error(fmt.Sprintf("Unable to unmarshal the future milestone field in database"), "err", err)

		return nil, nil, fmt.Errorf("%w(%v) for future milestone field, data %v(%q)",
			ErrIncorrectFutureMilestoneField, err, data, string(data))
	}

	order, list := futureMilestoneField.Order, futureMilestoneField.List

	return order, list, nil
}
