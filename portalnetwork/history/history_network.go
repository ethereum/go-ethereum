package history

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/view"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/portalnetwork/portalwire"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type ContentType byte

const (
	BlockHeaderType       ContentType = 0x00
	BlockBodyType         ContentType = 0x01
	ReceiptsType          ContentType = 0x02
	BlockHeaderNumberType ContentType = 0x03
	// EpochAccumulatorType ContentType = 0x03
)

var (
	ErrWithdrawalHashIsNotEqual = errors.New("withdrawals hash is not equal")
	ErrTxHashIsNotEqual         = errors.New("tx hash is not equal")
	ErrUnclesHashIsNotEqual     = errors.New("uncles hash is not equal")
	ErrReceiptsHashIsNotEqual   = errors.New("receipts hash is not equal")
	ErrContentOutOfRange        = errors.New("content out of range")
	ErrHeaderWithProofIsInvalid = errors.New("header proof is invalid")
	ErrInvalidBlockHash         = errors.New("invalid block hash")
	ErrInvalidBlockNumber       = errors.New("invalid block number")
)

var emptyReceiptHash = hexutil.MustDecode("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

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

type Network struct {
	portalProtocol    *portalwire.PortalProtocol
	masterAccumulator *MasterAccumulator
	closeCtx          context.Context
	closeFunc         context.CancelFunc
	log               log.Logger
}

func NewHistoryNetwork(portalProtocol *portalwire.PortalProtocol, accu *MasterAccumulator) *Network {
	ctx, cancel := context.WithCancel(context.Background())

	return &Network{
		portalProtocol:    portalProtocol,
		masterAccumulator: accu,
		closeCtx:          ctx,
		closeFunc:         cancel,
		log:               log.New("sub-protocol", "history"),
	}
}

func (h *Network) Start() error {
	err := h.portalProtocol.Start()
	if err != nil {
		return err
	}
	go h.processContentLoop(h.closeCtx)
	h.log.Debug("history network start successfully")
	return nil
}

func (h *Network) Stop() {
	h.closeFunc()
	h.portalProtocol.Stop()
}

// Currently doing 4 retries on lookups but only when the validation fails.
const requestRetries = 4

func (h *Network) GetBlockHeader(blockHash []byte) (*types.Header, error) {
	contentKey := newContentKey(BlockHeaderType, blockHash).encode()
	contentId := h.portalProtocol.ToContentId(contentKey)
	h.log.Trace("contentKey convert to contentId", "contentKey", hexutil.Encode(contentKey), "contentId", hexutil.Encode(contentId))
	if !h.portalProtocol.InRange(contentId) {
		return nil, ErrContentOutOfRange
	}

	res, err := h.portalProtocol.Get(contentKey, contentId)
	// other error
	if err != nil && !errors.Is(err, storage.ErrContentNotFound) {
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
		content, _, err := h.portalProtocol.ContentLookup(contentKey, contentId)
		if err != nil {
			h.log.Error("getBlockHeader failed", "contentKey", hexutil.Encode(contentKey), "err", err)
			continue
		}

		headerWithProof, err := DecodeBlockHeaderWithProof(content)
		if err != nil {
			h.log.Error("decodeBlockHeaderWithProof failed", "content", hexutil.Encode(content), "err", err)
			continue
		}

		header, err := ValidateBlockHeaderBytes(headerWithProof.Header, blockHash)
		if err != nil {
			h.log.Error("validateBlockHeaderBytes failed", "header", hexutil.Encode(headerWithProof.Header), "blockhash", hexutil.Encode(blockHash), "err", err)
			continue
		}
		valid, err := h.verifyHeader(header, *headerWithProof.Proof)
		if err != nil || !valid {
			h.log.Error("verifyHeader failed", "err", err)
			continue
		}
		err = h.portalProtocol.Put(contentKey, contentId, content)
		if err != nil {
			h.log.Error("failed to store content in getBlockHeader", "contentKey", hexutil.Encode(contentKey), "content", hexutil.Encode(content))
		}
		return header, nil
	}
	return nil, storage.ErrContentNotFound
}

func (h *Network) GetBlockBody(blockHash []byte) (*types.Body, error) {
	header, err := h.GetBlockHeader(blockHash)
	if err != nil {
		return nil, err
	}
	contentKey := newContentKey(BlockBodyType, blockHash).encode()
	contentId := h.portalProtocol.ToContentId(contentKey)

	if !h.portalProtocol.InRange(contentId) {
		return nil, ErrContentOutOfRange
	}

	res, err := h.portalProtocol.Get(contentKey, contentId)
	// other error
	// TODO maybe use nil res to replace the ErrContentNotFound
	if err != nil && !errors.Is(err, storage.ErrContentNotFound) {
		return nil, err
	}
	// no error
	if err == nil {
		body, err := DecodePortalBlockBodyBytes(res)
		return body, err
	}
	// no content in local storage

	for retries := 0; retries < requestRetries; retries++ {
		content, _, err := h.portalProtocol.ContentLookup(contentKey, contentId)
		if err != nil {
			h.log.Error("getBlockBody failed", "contentKey", hexutil.Encode(contentKey), "err", err)
			continue
		}
		body, err := DecodePortalBlockBodyBytes(content)
		if err != nil {
			h.log.Error("decodePortalBlockBodyBytes failed", "content", hexutil.Encode(content), "err", err)
			continue
		}

		err = validateBlockBody(body, header)
		if err != nil {
			h.log.Error("validateBlockBody failed", "header", "err", err)
			continue
		}
		err = h.portalProtocol.Put(contentKey, contentId, content)
		if err != nil {
			h.log.Error("failed to store content in getBlockBody", "contentKey", hexutil.Encode(contentKey), "content", hexutil.Encode(content))
		}
		return body, nil
	}
	return nil, storage.ErrContentNotFound
}

func (h *Network) GetReceipts(blockHash []byte) ([]*types.Receipt, error) {
	header, err := h.GetBlockHeader(blockHash)
	if err != nil {
		return nil, err
	}
	contentKey := newContentKey(ReceiptsType, blockHash).encode()
	contentId := h.portalProtocol.ToContentId(contentKey)

	if !h.portalProtocol.InRange(contentId) {
		return nil, ErrContentOutOfRange
	}

	res, err := h.portalProtocol.Get(contentKey, contentId)
	// other error
	if err != nil && !errors.Is(err, storage.ErrContentNotFound) {
		return nil, err
	}
	// no error
	if err == nil {
		portalReceipte := new(PortalReceipts)
		err := portalReceipte.UnmarshalSSZ(res)
		if err != nil {
			return nil, err
		}
		receipts, err := FromPortalReceipts(portalReceipte)
		return receipts, err
	}
	// no content in local storage

	for retries := 0; retries < requestRetries; retries++ {
		content, _, err := h.portalProtocol.ContentLookup(contentKey, contentId)
		if err != nil {
			h.log.Error("getReceipts failed", "contentKey", hexutil.Encode(contentKey), "err", err)
			continue
		}
		receipts, err := ValidatePortalReceiptsBytes(content, header.ReceiptHash.Bytes())
		if err != nil {
			h.log.Error("getReceipts failed", "err", err)
			continue
		}
		err = h.portalProtocol.Put(contentKey, contentId, content)
		if err != nil {
			h.log.Error("failed to store content in getReceipts", "contentKey", hexutil.Encode(contentKey), "content", hexutil.Encode(content))
		}
		return receipts, nil
	}
	return nil, storage.ErrContentNotFound
}

func (h *Network) verifyHeader(header *types.Header, proof BlockHeaderProof) (bool, error) {
	return h.masterAccumulator.VerifyHeader(*header, proof)
}

func ValidateBlockBodyBytes(bodyBytes []byte, header *types.Header) (*types.Body, error) {
	// TODO check shanghai, pos and legacy block
	body, err := DecodePortalBlockBodyBytes(bodyBytes)
	if err != nil {
		return nil, err
	}
	err = validateBlockBody(body, header)
	return body, err
}

func DecodePortalBlockBodyBytes(bodyBytes []byte) (*types.Body, error) {
	blockBodyShanghai := new(PortalBlockBodyShanghai)
	err := blockBodyShanghai.UnmarshalSSZ(bodyBytes)
	if err == nil {
		return FromPortalBlockBodyShanghai(blockBodyShanghai)
	}

	blockBodyLegacy := new(BlockBodyLegacy)
	err = blockBodyLegacy.UnmarshalSSZ(bodyBytes)
	if err == nil {
		return FromBlockBodyLegacy(blockBodyLegacy)
	}
	return nil, errors.New("all portal block body decodings failed")
}

func validateBlockBody(body *types.Body, header *types.Header) error {
	if hash := types.CalcUncleHash(body.Uncles); !bytes.Equal(hash[:], header.UncleHash.Bytes()) {
		return ErrUnclesHashIsNotEqual
	}

	if hash := types.DeriveSha(types.Transactions(body.Transactions), trie.NewStackTrie(nil)); !bytes.Equal(hash[:], header.TxHash.Bytes()) {
		return ErrTxHashIsNotEqual
	}
	if body.Withdrawals == nil {
		return nil
	}
	if hash := types.DeriveSha(types.Withdrawals(body.Withdrawals), trie.NewStackTrie(nil)); !bytes.Equal(hash[:], header.WithdrawalsHash.Bytes()) {
		return ErrWithdrawalHashIsNotEqual
	}
	return nil
}

// EncodeBlockBody encode types.Body to ssz bytes
func EncodeBlockBody(body *types.Body) ([]byte, error) {
	if len(body.Withdrawals) > 0 {
		blockShanghai, err := toPortalBlockBodyShanghai(body)
		if err != nil {
			return nil, err
		}
		return blockShanghai.MarshalSSZ()
	} else {
		legacyBlock, err := toBlockBodyLegacy(body)
		if err != nil {
			return nil, err
		}
		return legacyBlock.MarshalSSZ()
	}
}

// toPortalBlockBodyShanghai convert types.Body to PortalBlockBodyShanghai
func toPortalBlockBodyShanghai(b *types.Body) (*PortalBlockBodyShanghai, error) {
	legacy, err := toBlockBodyLegacy(b)
	if err != nil {
		return nil, err
	}
	withdrawals := make([][]byte, 0, len(b.Withdrawals))
	for _, w := range b.Withdrawals {
		b, err := rlp.EncodeToBytes(w)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, b)
	}
	return &PortalBlockBodyShanghai{Transactions: legacy.Transactions, Uncles: legacy.Uncles, Withdrawals: withdrawals}, nil
}

// toBlockBodyLegacy convert types.Body to BlockBodyLegacy
func toBlockBodyLegacy(b *types.Body) (*BlockBodyLegacy, error) {
	txs := make([][]byte, 0, len(b.Transactions))

	for _, tx := range b.Transactions {
		txBytes, err := rlp.EncodeToBytes(tx)
		if err != nil {
			return nil, err
		}
		txs = append(txs, txBytes)
	}

	uncleBytes, err := rlp.EncodeToBytes(b.Uncles)
	if err != nil {
		return nil, err
	}
	return &BlockBodyLegacy{Uncles: uncleBytes, Transactions: txs}, err
}

// FromPortalBlockBodyShanghai convert PortalBlockBodyShanghai to types.Body
func FromPortalBlockBodyShanghai(b *PortalBlockBodyShanghai) (*types.Body, error) {
	transactions := make([]*types.Transaction, 0, len(b.Transactions))
	for _, t := range b.Transactions {
		tran := new(types.Transaction)
		err := tran.UnmarshalBinary(t)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tran)
	}
	uncles := make([]*types.Header, 0, len(b.Uncles))
	err := rlp.DecodeBytes(b.Uncles, &uncles)
	withdrawals := make([]*types.Withdrawal, 0, len(b.Withdrawals))
	for _, w := range b.Withdrawals {
		withdrawal := new(types.Withdrawal)
		err := rlp.DecodeBytes(w, withdrawal)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return &types.Body{
		Uncles:       uncles,
		Transactions: transactions,
		Withdrawals:  withdrawals,
	}, err
}

// FromBlockBodyLegacy convert BlockBodyLegacy to types.Body
func FromBlockBodyLegacy(b *BlockBodyLegacy) (*types.Body, error) {
	transactions := make([]*types.Transaction, 0, len(b.Transactions))
	for _, t := range b.Transactions {
		tran := new(types.Transaction)
		err := tran.UnmarshalBinary(t)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tran)
	}
	uncles := make([]*types.Header, 0, len(b.Uncles))
	err := rlp.DecodeBytes(b.Uncles, &uncles)
	return &types.Body{
		Uncles:       uncles,
		Transactions: transactions,
	}, err
}

// FromPortalReceipts convert PortalReceipts to types.Receipt
func FromPortalReceipts(r *PortalReceipts) ([]*types.Receipt, error) {
	res := make([]*types.Receipt, 0, len(r.Receipts))
	for _, reci := range r.Receipts {
		recipt := new(types.Receipt)
		err := recipt.UnmarshalBinary(reci)
		if err != nil {
			return nil, err
		}
		res = append(res, recipt)
	}
	return res, nil
}

func ValidatePortalReceiptsBytes(receiptBytes, receiptsRoot []byte) ([]*types.Receipt, error) {
	portalReceipts := new(PortalReceipts)
	err := portalReceipts.UnmarshalSSZ(receiptBytes)
	if err != nil {
		return nil, err
	}

	receipts, err := FromPortalReceipts(portalReceipts)
	if err != nil {
		return nil, err
	}

	root := types.DeriveSha(types.Receipts(receipts), trie.NewStackTrie(nil))

	if !bytes.Equal(root[:], receiptsRoot) {
		return nil, errors.New("receipt root is not equal to the header.ReceiptHash")
	}
	return receipts, nil
}

func EncodeReceipts(receipts []*types.Receipt) ([]byte, error) {
	portalReceipts, err := ToPortalReceipts(receipts)
	if err != nil {
		return nil, err
	}
	return portalReceipts.MarshalSSZ()
}

// ToPortalReceipts convert types.Receipt to PortalReceipts
func ToPortalReceipts(receipts []*types.Receipt) (*PortalReceipts, error) {
	res := make([][]byte, 0, len(receipts))
	for _, r := range receipts {
		b, err := r.MarshalBinary()
		if err != nil {
			return nil, err
		}
		res = append(res, b)
	}
	return &PortalReceipts{Receipts: res}, nil
}

func (h *Network) processContentLoop(ctx context.Context) {
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

func (h *Network) validateContent(contentKey []byte, content []byte) error {
	switch ContentType(contentKey[0]) {
	case BlockHeaderType:
		headerWithProof, err := DecodeBlockHeaderWithProof(content)
		if err != nil {
			return err
		}
		header, err := DecodeBlockHeader(headerWithProof.Header)
		if err != nil {
			return err
		}
		if !bytes.Equal(header.Hash().Bytes(), contentKey[1:]) {
			return ErrInvalidBlockHash
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
		header, err := h.GetBlockHeader(contentKey[1:])
		if err != nil {
			return err
		}
		_, err = ValidateBlockBodyBytes(content, header)
		return err
	case ReceiptsType:
		header, err := h.GetBlockHeader(contentKey[1:])
		if err != nil {
			return err
		}
		if bytes.Equal(header.ReceiptHash.Bytes(), emptyReceiptHash) {
			if len(content) > 0 {
				return fmt.Errorf("content should be empty, but received %v", content)
			}
			return nil
		}
		_, err = ValidatePortalReceiptsBytes(content, header.ReceiptHash.Bytes())
		return err
	case BlockHeaderNumberType:
		headerWithProof, err := DecodeBlockHeaderWithProof(content)
		if err != nil {
			return err
		}
		header, err := DecodeBlockHeader(headerWithProof.Header)
		if err != nil {
			return err
		}
		blockNumber := view.Uint64View(0)
		err = blockNumber.Deserialize(codec.NewDecodingReader(bytes.NewReader(contentKey[1:]), uint64(len(contentKey[1:]))))
		if err != nil {
			return err
		}
		if header.Number.Cmp(big.NewInt(int64(blockNumber))) != 0 {
			return ErrInvalidBlockNumber
		}
		valid, err := h.verifyHeader(header, *headerWithProof.Proof)
		if err != nil {
			return err
		}
		if !valid {
			return ErrHeaderWithProofIsInvalid
		}
		return err
	}
	return errors.New("unknown content type")
}

func (h *Network) validateContents(contentKeys [][]byte, contents [][]byte) error {
	for i, content := range contents {
		contentKey := contentKeys[i]
		err := h.validateContent(contentKey, content)
		if err != nil {
			h.log.Error("content validate failed", "contentKey", hexutil.Encode(contentKey), "content", hexutil.Encode(content), "err", err)
			return fmt.Errorf("content validate failed with content key %x and content %x", contentKey, content)
		}
		contentId := h.portalProtocol.ToContentId(contentKey)
		_ = h.portalProtocol.Put(contentKey, contentId, content)
	}
	return nil
}

func ValidateBlockHeaderBytes(headerBytes []byte, blockHash []byte) (*types.Header, error) {
	header := new(types.Header)
	err := rlp.DecodeBytes(headerBytes, header)
	if err != nil {
		return nil, err
	}
	hash := header.Hash()
	if !bytes.Equal(hash[:], blockHash) {
		return nil, ErrInvalidBlockHash
	}
	return header, nil
}

func DecodeBlockHeader(headerBytes []byte) (*types.Header, error) {
	header := new(types.Header)
	err := rlp.DecodeBytes(headerBytes, header)
	if err != nil {
		return nil, err
	}
	return header, nil
}

func DecodeBlockHeaderWithProof(content []byte) (*BlockHeaderWithProof, error) {
	headerWithProof := new(BlockHeaderWithProof)
	err := headerWithProof.UnmarshalSSZ(content)
	return headerWithProof, err
}

func decodeEpochAccumulator(data []byte) (*EpochAccumulator, error) {
	epochAccu := new(EpochAccumulator)
	err := epochAccu.UnmarshalSSZ(data)
	return epochAccu, err
}
