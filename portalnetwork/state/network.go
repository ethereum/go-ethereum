package state

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/portalnetwork/history"
	"github.com/ethereum/go-ethereum/portalnetwork/portalwire"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/codec"
)

type StateNetwork struct {
	portalProtocol *portalwire.PortalProtocol
	closeCtx       context.Context
	closeFunc      context.CancelFunc
	log            log.Logger
	spec           *common.Spec
	client         *rpc.Client
}

func NewStateNetwork(portalProtocol *portalwire.PortalProtocol, client *rpc.Client) *StateNetwork {
	ctx, cancel := context.WithCancel(context.Background())
	return &StateNetwork{
		portalProtocol: portalProtocol,
		closeCtx:       ctx,
		closeFunc:      cancel,
		log:            log.New("sub-protocol", "state"),
		spec:           configs.Mainnet,
		client:         client,
	}
}

func (h *StateNetwork) Start() error {
	err := h.portalProtocol.Start()
	if err != nil {
		return err
	}
	go h.processContentLoop(h.closeCtx)
	h.log.Debug("state network start successfully")
	return nil
}

func (h *StateNetwork) Stop() {
	h.closeFunc()
	h.portalProtocol.Stop()
}

func (h *StateNetwork) processContentLoop(ctx context.Context) {
	contentChan := h.portalProtocol.GetContent()
	for {
		select {
		case <-ctx.Done():
			return
		case contentElement := <-contentChan:
			err := h.validateContents(contentElement.ContentKeys, contentElement.Contents)
			if err != nil {
				h.log.Error("validate content failed", "err", err)
				continue
			}

			go func(ctx context.Context) {
				select {
				case <-ctx.Done():
					return
				default:
					var gossippedNum int
					gossippedNum, err = h.portalProtocol.Gossip(&contentElement.Node, contentElement.ContentKeys, contentElement.Contents)
					h.log.Trace("gossippedNum", "gossippedNum", gossippedNum)
					if err != nil {
						h.log.Error("gossip failed", "err", err)
						return
					}
				}
			}(ctx)
		}
	}
}

func (h *StateNetwork) validateContents(contentKeys [][]byte, contents [][]byte) error {
	for i, content := range contents {
		contentKey := contentKeys[i]
		err := h.validateContent(contentKey, content)
		if err != nil {
			h.log.Error("content validate failed", "contentKey", hexutil.Encode(contentKey), "content", hexutil.Encode(content), "err", err)
			return fmt.Errorf("content validate failed with content key %x and content %x", contentKey, content)
		}
		contentId := h.portalProtocol.ToContentId(contentKey)
		err = h.portalProtocol.Put(contentKey, contentId, content)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *StateNetwork) validateContent(contentKey []byte, content []byte) error {
	keyType := contentKey[0]
	switch keyType {
	case AccountTrieNodeType:
		return h.validateAccountTrieNode(contentKey[1:], content)
	case ContractStorageTrieNodeType:
		return h.validateContractStorageTrieNode(contentKey[1:], content)
	case ContractByteCodeType:
		return h.validateContractByteCode(contentKey[1:], content)
	}
	return errors.New("unknown content type")
}

func (h *StateNetwork) validateAccountTrieNode(contentKey []byte, content []byte) error {
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
	// get HeaderWithProof in history network
	stateRoot, err := h.getStateRoot(accountData.BlockHash)

	if err != nil {
		return err
	}
	err = validateNodeTrieProof(stateRoot, accountKey.NodeHash, &accountKey.Path, &accountData.Proof)
	return err
}

func (h *StateNetwork) validateContractStorageTrieNode(contentKey []byte, content []byte) error {
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
	stateRoot, err := h.getStateRoot(contractProof.BlockHash)

	if err != nil {
		return err
	}

	accountState, err := validateAccountState(stateRoot, contractStorageKey.AddressHash, &contractProof.AccountProof)
	if err != nil {
		return err
	}
	err = validateNodeTrieProof(common.Bytes32(accountState.Root), contractStorageKey.NodeHash, &contractStorageKey.Path, &contractProof.StoregeProof)
	return err
}

func (h *StateNetwork) validateContractByteCode(contentKey []byte, content []byte) error {
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
	stateRoot, err := h.getStateRoot(contractBytecodeWithProof.BlockHash)
	if err != nil {
		return err
	}
	accountState, err := validateAccountState(stateRoot, contractByteCodeKey.AddressHash, &contractBytecodeWithProof.AccountProof)
	if err != nil {
		return err
	}
	if !bytes.Equal(accountState.CodeHash, contractByteCodeKey.CodeHash[:]) {
		return errors.New("account state is invalid")
	}
	return nil
}

func (h *StateNetwork) getStateRoot(blockHash common.Bytes32) (common.Bytes32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	contentKey := make([]byte, 0)
	contentKey = append(contentKey, byte(history.BlockHeaderType))
	contentKey = append(contentKey, blockHash[:]...)

	arg := hexutil.Encode(contentKey)
	res := &portalwire.ContentInfo{}
	err := h.client.CallContext(ctx, res, "portal_historyGetContent", arg)
	if err != nil {
		return common.Bytes32{}, err
	}
	data, err := hexutil.Decode(res.Content)
	if err != nil {
		return common.Bytes32{}, err
	}
	headerWithProof, err := history.DecodeBlockHeaderWithProof(data)
	if err != nil {
		return common.Bytes32{}, err
	}
	header := new(types.Header)
	err = rlp.DecodeBytes(headerWithProof.Header, header)
	if err != nil {
		return common.Bytes32{}, err
	}
	return common.Bytes32(header.Root), nil
}

func validateNodeTrieProof(rootHash common.Bytes32, nodeHash common.Bytes32, path *Nibbles, proof *TrieProof) error {
	lastNode, p, err := validateTrieProof(rootHash, path.Nibbles, proof)
	if err != nil {
		return err
	}
	if len(p) != 0 {
		return errors.New("path is too long")
	}
	err = checkNodeHash(&lastNode, nodeHash[:])
	if err != nil {
		return err
	}
	return nil
}

func validateAccountState(rootHash common.Bytes32, addrrssHash common.Bytes32, proof *TrieProof) (*types.StateAccount, error) {
	path := make([]byte, 0, len(addrrssHash)*2)
	for _, item := range addrrssHash {
		before, after := unpackNibblePair(item)
		path = append(path, before, after)
	}
	lastProof, p, err := validateTrieProof(rootHash, path, proof)
	if err != nil {
		return nil, err
	}
	n, err := trie.DecodeTrieNode(nil, lastProof)
	if err != nil {
		return nil, err
	}
	stateBytes, _, err := trie.TraverseTrieNode(n, p)
	if err != nil {
		return nil, err
	}
	return types.FullAccount(stateBytes)
}

func validateTrieProof(rootHash common.Bytes32, path []byte, proof *TrieProof) (EncodedTrieNode, []byte, error) {
	if len(*proof) == 0 {
		return nil, nil, errors.New("proof should be empty")
	}
	firstNode := []EncodedTrieNode(*proof)[0]
	err := checkNodeHash(&firstNode, rootHash[:])
	if err != nil {
		return nil, nil, err
	}

	node := firstNode
	remainingPath := path

	for _, nextNode := range []EncodedTrieNode(*proof)[1:] {
		n, err := trie.DecodeTrieNode(nil, node)
		if err != nil {
			return nil, nil, err
		}
		hashNode, p, err := trie.TraverseTrieNode(n, remainingPath)
		if err != nil {
			return nil, nil, err
		}
		err = checkNodeHash(&nextNode, hashNode)

		if err != nil {
			return nil, nil, err
		}

		node = nextNode
		remainingPath = p
	}
	return node, remainingPath, nil
}

func checkNodeHash(node *EncodedTrieNode, hash []byte) error {
	nodeHash := node.NodeHash()
	if !bytes.Equal(nodeHash[:], hash[:]) {
		return fmt.Errorf("node hash is not equal, expect: %v, actual: %v", hash, nodeHash)
	}
	return nil
}
