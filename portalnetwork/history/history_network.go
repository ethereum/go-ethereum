package history

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/ethereum/go-ethereum/rlp"
)

type ContentType byte

const (
	BlockHeaderType      ContentType = 0x00
	BlockBodyType        ContentType = 0x01
	ReceiptsType         ContentType = 0x02
	EpochAccumulatorType ContentType = 0x03
)

var (
	ErrWithdrawalHashIsNotEqual = errors.New("withdrawals hash is not equal")
	ErrTxHashIsNotEqual         = errors.New("tx hash is not equal")
	ErrUnclesHashIsNotEqual     = errors.New("uncles hash is not equal")
	ErrReceiptsHashIsNotEqual   = errors.New("receipts hash is not equal")
	ErrContentOutOfRange        = errors.New("content out of range")
	ErrHeaderWithProofIsInvalid = errors.New("header proof is invalid")
	ErrInvalidBlockHash         = errors.New("invalid block hash")
)

type ContentKey struct {
	selector ContentType
	data     []byte
}

func newContentKey(selector ContentType, hash []byte) *ContentKey {
	return &ContentKey{
		selector: selector,
		data:     hash,
	}
}

func (c *ContentKey) encode() []byte {
	res := make([]byte, 0, len(c.data)+1)
	res = append(res, byte(c.selector))
	res = append(res, c.data...)
	return res
}

type HistoryNetwork struct {
	portalProtocol    *discover.PortalProtocol
	masterAccumulator *MasterAccumulator
}

func NewHistoryNetwork(portalProtocol *discover.PortalProtocol, accu *MasterAccumulator) *HistoryNetwork {
	return &HistoryNetwork{
		portalProtocol:    portalProtocol,
		masterAccumulator: accu,
	}
}

func (h *HistoryNetwork) Start() error {
	err := h.portalProtocol.Start()
	if err != nil {
		return err
	}
	go h.processContentLoop()
	return nil
}

// Currently doing 4 retries on lookups but only when the validation fails.
const requestRetries = 4

func (h *HistoryNetwork) GetBlockHeader(blockHash []byte) (*types.Header, error) {
	contentKey := newContentKey(BlockHeaderType, blockHash).encode()
	contentId := h.portalProtocol.ToContentId(contentKey)
	if !h.portalProtocol.InRange(contentId) {
		return nil, ErrContentOutOfRange
	}

	res, err := h.portalProtocol.Get(contentId)
	// other error
	if err != nil && err != storage.ErrContentNotFound {
		return nil, err
	}
	// no error
	if err == nil {
		blockHeaderWithProof, err := DecodeBlockHeaderWithProof(res)
		if err != nil {
			return nil, err
		}
		header := new(types.Header)
		err = rlp.DecodeBytes(blockHeaderWithProof.Header, header)
		return header, err
	}
	// no content in local storage
	for retries := 0; retries < requestRetries; retries++ {
		// TODO log the err and continue
		content, err := h.portalProtocol.ContentLookup(contentKey)
		if err != nil {
			continue
		}

		headerWithProof, err := DecodeBlockHeaderWithProof(content)
		if err != nil {
			continue
		}

		header, err := ValidateBlockHeaderBytes(headerWithProof.Header, blockHash)
		if err != nil {
			continue
		}
		valid, err := h.verifyHeader(header, *headerWithProof.Proof)
		if err != nil || !valid {
			continue
		}
		// TODO handle the error
		_ = h.portalProtocol.Put(contentId, content)
		return header, nil
	}
	return nil, storage.ErrContentNotFound
}

func (h *HistoryNetwork) verifyHeader(header *types.Header, proof BlockHeaderProof) (bool, error) {
	return h.masterAccumulator.VerifyHeader(*header, proof)
}

func (h *HistoryNetwork) processContentLoop() {
	contentChan := h.portalProtocol.GetContent()
	for contentElement := range contentChan {
		err := h.validateContents(contentElement.ContentKeys, contentElement.Contents)
		if err != nil {
			continue
		}
		// TODO gossip the validate content
	}
}

func (h *HistoryNetwork) validateContent(contentKey []byte, content []byte) error {
	switch ContentType(contentKey[0]) {
	case BlockHeaderType:
		headerWithProof, err := DecodeBlockHeaderWithProof(content)
		if err != nil {
			return err
		}
		header, err := ValidateBlockHeaderBytes(headerWithProof.Header, contentKey[1:])
		if err != nil {
			return err
		}
		valid, err := h.verifyHeader(header, *headerWithProof.Proof)
		if err != nil {
			return err
		}
		if !valid {
			return ErrHeaderWithProofIsInvalid
		}
		return err
	case BlockBodyType:
		// TODO
	case ReceiptsType:
		// TODO
	case EpochAccumulatorType:
		// TODO
	}
	return errors.New("unknown content type")
}

func (h *HistoryNetwork) validateContents(contentKeys [][]byte, contents [][]byte) error {
	for i, content := range contents {
		contentKey := contentKeys[i]
		err := h.validateContent(contentKey, content)
		if err != nil {
			return fmt.Errorf("content validate failed with content key %v", contentKey)
		}
		contentId := h.portalProtocol.ToContentId(contentKey)
		_ = h.portalProtocol.Put(contentId, content)
	}
	return nil
}

func ValidateBlockHeaderBytes(headerBytes []byte, blockHash []byte) (*types.Header, error) {
	header := new(types.Header)
	err := rlp.DecodeBytes(headerBytes, header)
	if err != nil {
		return nil, err
	}
	if header.ExcessBlobGas != nil {
		return nil, errors.New("EIP-4844 not yet implemented")
	}
	hash := header.Hash()
	if !bytes.Equal(hash[:], blockHash) {
		return nil, ErrInvalidBlockHash
	}
	return header, nil
}

func DecodeBlockHeaderWithProof(content []byte) (*BlockHeaderWithProof, error) {
	headerWithProof := new(BlockHeaderWithProof)
	err := headerWithProof.UnmarshalSSZ(content)
	return headerWithProof, err
}
