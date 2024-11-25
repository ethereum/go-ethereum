package state

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/portalnetwork/portalwire"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/holiman/uint256"
	"github.com/protolambda/ztyp/codec"
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

var portalStorageMetrics *portalwire.PortalStorageMetrics

func NewStateStorage(store storage.ContentStorage, db *sql.DB) *StateStorage {
	storage := &StateStorage{
		store: store,
		db:    db,
		log:   log.New("storage", "state"),
	}

	var err error
	portalStorageMetrics, err = portalwire.NewPortalStorageMetrics("state", db)
	if err != nil {
		return nil
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
		portalStorageMetrics.EntriesCount.Inc(1)
		portalStorageMetrics.ContentStorageUsage.Inc(int64(len(content)))
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
		portalStorageMetrics.EntriesCount.Inc(1)
		portalStorageMetrics.ContentStorageUsage.Inc(int64(len(content)))
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
		portalStorageMetrics.EntriesCount.Inc(1)
		portalStorageMetrics.ContentStorageUsage.Inc(int64(len(content)))
	}
	return nil
}
