package state

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/holiman/uint256"
	"github.com/protolambda/ztyp/codec"
)

var (
	radiusRatio         metrics.GaugeFloat64
	entriesCount        metrics.Gauge
	contentStorageUsage metrics.Gauge
)

const (
	countEntrySql         = "SELECT COUNT(1) FROM kvstore;"
	conentStorageUsageSql = "SELECT SUM( length(value) ) FROM kvstore;"
)

func defaultContentIdFunc(contentKey []byte) []byte {
	digest := sha256.Sum256(contentKey)
	return digest[:]
}

var _ storage.ContentStorage = &StateStorage{}

type StateStorage struct {
	store storage.ContentStorage
	db    *sql.DB
	log   log.Logger
}

func NewStateStorage(store storage.ContentStorage, db *sql.DB) *StateStorage {
	storage := &StateStorage{
		store: store,
		db:    db,
		log:   log.New("storage", "state"),
	}

	if metrics.Enabled {
		radiusRatio = metrics.NewRegisteredGaugeFloat64("portal/state/radius_ratio", nil)
		radiusRatio.Update(1)

		entriesCount = metrics.NewRegisteredGauge("portal/state/entry_count", nil)
		log.Info("Counting entities in state storage for metrics")
		count, err := db.Prepare(countEntrySql)
		if err != nil {
			log.Error("Querry preparation error", "network", "state", "metric", "entry_count", "err", err)
			return nil
		}
		var res *int64 = new(int64)
		q := count.QueryRow()
		if q.Err() != nil {
			log.Error("Querry execution error", "network", "state", "metric", "entry_count", "err", err)
			return nil
		} else {
			q.Scan(&res)
		}
		entriesCount.Update(*res)

		contentStorageUsage = metrics.NewRegisteredGauge("portal/state/content_storage", nil)
		log.Info("Counting storage usage (bytes) in state for metrics")
		str, err := db.Prepare(conentStorageUsageSql)
		if err != nil {
			log.Error("Querry preparation error", "network", "state", "metric", "content_storage", "err", err)
			return nil
		}
		var resStr *int64 = new(int64)
		q = str.QueryRow()
		if q.Err() != nil {
			log.Error("Querry execution error", "network", "state", "metric", "content_storage", "err", err)
			return nil
		} else {
			q.Scan(resStr)
		}
		contentStorageUsage.Update(int64(*resStr))
	}

	return storage
}

// Get implements storage.ContentStorage.
func (s *StateStorage) Get(contentKey []byte, contentId []byte) ([]byte, error) {
	return s.store.Get(contentKey, contentId)
}

// Put implements storage.ContentStorage.
func (s *StateStorage) Put(contentKey []byte, contentId []byte, content []byte) error {
	keyType := contentKey[0]
	switch keyType {
	case AccountTrieNodeType:
		return s.putAccountTrieNode(contentKey[1:], contentId, content)
	case ContractStorageTrieNodeType:
		return s.putContractStorageTrieNode(contentKey[1:], contentId, content)
	case ContractByteCodeType:
		return s.putContractBytecode(contentKey[1:], contentId, content)
	}
	return errors.New("unknown content type")
}

// Radius implements storage.ContentStorage.
func (s *StateStorage) Radius() *uint256.Int {
	return s.store.Radius()
}

func (s *StateStorage) putAccountTrieNode(contentKey []byte, contentId []byte, content []byte) error {
	accountKey := &AccountTrieNodeKey{}
	err := accountKey.Deserialize(codec.NewDecodingReader(bytes.NewReader(contentKey), uint64(len(contentKey))))
	if err != nil {
		return err
	}
	accountData := &AccountTrieNodeWithProof{}
	err = accountData.Deserialize(codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
	if err != nil {
		return err
	}
	length := len(accountData.Proof)
	lastProof := accountData.Proof[length-1]

	lastNodeHash := crypto.Keccak256(lastProof)
	if !bytes.Equal(lastNodeHash, accountKey.NodeHash[:]) {
		return errors.New("hash of the trie node doesn't match key's node_hash")
	}
	lastTrieNode := &TrieNode{
		Node: lastProof,
	}
	var contentValueBuf bytes.Buffer
	err = lastTrieNode.Serialize(codec.NewEncodingWriter(&contentValueBuf))
	if err != nil {
		return err
	}
	err = s.store.Put(contentId, contentId, contentValueBuf.Bytes())
	if err != nil {
		s.log.Error("failed to save data after validate", "type", contentKey[0], "key", contentKey[1:], "value", content)
	} else if metrics.Enabled {
		entriesCount.Inc(1)
		contentStorageUsage.Inc(int64(len(content)))
	}
	return nil
}

func (s *StateStorage) putContractStorageTrieNode(contentKey []byte, contentId []byte, content []byte) error {
	contractStorageKey := &ContractStorageTrieNodeKey{}
	err := contractStorageKey.Deserialize(codec.NewDecodingReader(bytes.NewReader(contentKey), uint64(len(contentKey))))
	if err != nil {
		return err
	}
	contractProof := &ContractStorageTrieNodeWithProof{}
	err = contractProof.Deserialize(codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
	if err != nil {
		return err
	}
	length := len(contractProof.StoregeProof)
	lastProof := contractProof.StoregeProof[length-1]

	lastNodeHash := crypto.Keccak256(lastProof)
	if !bytes.Equal(lastNodeHash, contractStorageKey.NodeHash[:]) {
		return errors.New("hash of the contract storage node doesn't match key's node hash")
	}

	lastTrieNode := &TrieNode{
		Node: lastProof,
	}
	var contentValueBuf bytes.Buffer
	err = lastTrieNode.Serialize(codec.NewEncodingWriter(&contentValueBuf))
	if err != nil {
		return err
	}
	err = s.store.Put(contentId, contentId, contentValueBuf.Bytes())
	if err != nil {
		s.log.Error("failed to save data after validate", "type", contentKey[0], "key", contentKey[1:], "value", content)
	} else if metrics.Enabled {
		entriesCount.Inc(1)
		contentStorageUsage.Inc(int64(len(content)))
	}
	return nil
}

func (s *StateStorage) putContractBytecode(contentKey []byte, contentId []byte, content []byte) error {
	contractByteCodeKey := &ContractBytecodeKey{}
	err := contractByteCodeKey.Deserialize(codec.NewDecodingReader(bytes.NewReader(contentKey), uint64(len(contentKey))))
	if err != nil {
		return err
	}
	contractBytecodeWithProof := &ContractBytecodeWithProof{}
	err = contractBytecodeWithProof.Deserialize(codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
	if err != nil {
		return err
	}
	codeHash := crypto.Keccak256(contractBytecodeWithProof.Code)
	if !bytes.Equal(codeHash, contractByteCodeKey.CodeHash[:]) {
		return errors.New("hash of the contract byte doesn't match key's code hash")
	}
	container := &ContractBytecodeContainer{
		Code: contractBytecodeWithProof.Code,
	}
	var contentValueBuf bytes.Buffer
	err = container.Serialize(codec.NewEncodingWriter(&contentValueBuf))
	if err != nil {
		return err
	}
	err = s.store.Put(contentId, contentId, contentValueBuf.Bytes())
	if err != nil {
		s.log.Error("failed to save data after validate", "type", contentKey[0], "key", contentKey[1:], "value", content)
	} else if metrics.Enabled {
		entriesCount.Inc(1)
		contentStorageUsage.Inc(int64(len(content)))
	}
	return nil
}
