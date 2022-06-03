package builder

import (
	"bytes"
	"encoding/json"
	"html/template"
	"math/big"
	"net/http"
	_ "os"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"

	"github.com/flashbots/go-boost-utils/bls"
	boostTypes "github.com/flashbots/go-boost-utils/types"
)

type PubkeyHex string

type ValidatorData struct {
	FeeRecipient boostTypes.Address `json:"feeRecipient"`
	GasLimit     uint64             `json:"gasLimit"`
	Timestamp    uint64             `json:"timestamp"`
}

type IBeaconClient interface {
	isValidator(pubkey PubkeyHex) bool
	getProposerForNextSlot(requestedSlot uint64) (PubkeyHex, error)
	onForkchoiceUpdate() (PubkeyHex, error)
}

type Backend struct {
	beaconClient IBeaconClient

	builderSecretKey            *bls.SecretKey
	builderPublicKey            boostTypes.PublicKey
	serializedBuilderPoolPubkey hexutil.Bytes
	builderSigningDomain        boostTypes.Domain
	proposerSigningDomain       boostTypes.Domain
	enableBeaconChecks          bool

	validatorsLock sync.RWMutex
	validators     map[PubkeyHex]ValidatorData

	bestDataLock sync.Mutex
	bestHeader   *boostTypes.ExecutionPayloadHeader
	bestPayload  *boostTypes.ExecutionPayload
	profit       *big.Int

	indexTemplate *template.Template
}

func NewBackend(sk *bls.SecretKey, bc IBeaconClient, builderSigningDomain boostTypes.Domain, proposerSigningDomain boostTypes.Domain, enableBeaconChecks bool) *Backend {
	pkBytes := bls.PublicKeyFromSecretKey(sk).Compress()
	pk := boostTypes.PublicKey{}
	pk.FromSlice(pkBytes)

	_, err := bc.onForkchoiceUpdate()
	if err != nil {
		log.Error("could not initialize beacon client", "err", err)
	}

	indexTemplate, err := parseIndexTemplate()
	if err != nil {
		log.Error("could not parse index template", "err", err)
		indexTemplate = nil
	}
	return &Backend{
		beaconClient:                bc,
		builderSecretKey:            sk,
		builderPublicKey:            pk,
		serializedBuilderPoolPubkey: pkBytes,

		builderSigningDomain:  builderSigningDomain,
		proposerSigningDomain: proposerSigningDomain,
		enableBeaconChecks:    enableBeaconChecks,
		validators:            make(map[PubkeyHex]ValidatorData),
		indexTemplate:         indexTemplate,
	}
}

func (b *Backend) handleIndex(w http.ResponseWriter, req *http.Request) {
	if b.indexTemplate == nil {
		http.Error(w, "not available", http.StatusInternalServerError)
	}

	b.validatorsLock.RLock()
	noValidators := len(b.validators)
	b.validatorsLock.RUnlock()

	header := b.bestHeader
	headerData, err := json.MarshalIndent(header, "", "  ")
	if err != nil {
		headerData = []byte{}
	}

	payload := b.bestPayload
	payloadData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		payloadData = []byte{}
	}

	statusData := struct {
		NoValidators int
		Header       string
		Blocks       string
	}{noValidators, string(headerData), string(payloadData)}

	if err := b.indexTemplate.Execute(w, statusData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (b *Backend) handleStatus(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type httpErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(httpErrorResp{code, message}); err != nil {
		http.Error(w, message, code)
	}
}

func (b *Backend) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
	payload := []boostTypes.SignedValidatorRegistration{}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		log.Error("could not decode payload", "err", err)
		respondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	for _, registerRequest := range payload {
		if len(registerRequest.Message.Pubkey) != 48 {
			respondError(w, http.StatusBadRequest, "invalid pubkey")
			return
		}

		if len(registerRequest.Signature) != 96 {
			respondError(w, http.StatusBadRequest, "invalid signature")
			return
		}

		ok, err := boostTypes.VerifySignature(registerRequest.Message, b.builderSigningDomain, registerRequest.Message.Pubkey[:], registerRequest.Signature[:])
		if !ok || err != nil {
			log.Error("error verifying signature", "err", err)
			respondError(w, http.StatusBadRequest, "invalid signature")
			return
		}

		// Do not check timestamp before signature, as it would leak validator data
		if registerRequest.Message.Timestamp > uint64(time.Now().Add(10*time.Second).Unix()) {
			respondError(w, http.StatusBadRequest, "invalid payload")
			return
		}
	}

	for _, registerRequest := range payload {
		pubkeyHex := PubkeyHex(registerRequest.Message.Pubkey.String())
		if !b.beaconClient.isValidator(pubkeyHex) {
			respondError(w, http.StatusBadRequest, "not a validator")
			return
		}
	}

	b.validatorsLock.Lock()
	defer b.validatorsLock.Unlock()

	for _, registerRequest := range payload {
		pubkeyHex := PubkeyHex(registerRequest.Message.Pubkey.String())
		if previousValidatorData, ok := b.validators[pubkeyHex]; ok {
			if registerRequest.Message.Timestamp <= previousValidatorData.Timestamp {
				respondError(w, http.StatusBadRequest, "invalid timestamp")
				return
			}
		}
	}

	for _, registerRequest := range payload {
		pubkeyHex := PubkeyHex(registerRequest.Message.Pubkey.String())
		b.validators[pubkeyHex] = ValidatorData{
			FeeRecipient: registerRequest.Message.FeeRecipient,
			GasLimit:     registerRequest.Message.GasLimit,
			Timestamp:    registerRequest.Message.Timestamp,
		}

		log.Info("registered validator", "pubkey", pubkeyHex, "data", b.validators[pubkeyHex])
	}

	w.WriteHeader(http.StatusOK)
}

func (b *Backend) handleGetHeader(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	slot, err := strconv.Atoi(vars["slot"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "incorrect slot")
		return
	}
	parentHashHex := vars["parent_hash"]
	pubkeyHex := PubkeyHex(vars["pubkey"])

	b.validatorsLock.RLock()
	if _, ok := b.validators[pubkeyHex]; !ok {
		log.Error("missing validator", "validators", b.validators, "provided", pubkeyHex)
		b.validatorsLock.RUnlock()
		respondError(w, http.StatusBadRequest, "unknown validator")
		return
	}
	b.validatorsLock.RUnlock()

	// Do not validate slot separately, it will create a race between slot update and proposer key
	if nextSlotProposer, err := b.beaconClient.getProposerForNextSlot(uint64(slot)); err != nil || nextSlotProposer != pubkeyHex {
		log.Error("getHeader requested for public key other than next slots proposer", "requested", pubkeyHex, "expected", nextSlotProposer)
		if b.enableBeaconChecks {
			respondError(w, http.StatusBadRequest, "unknown validator")
			return
		}
	}

	b.bestDataLock.Lock()
	bestHeader := b.bestHeader
	profit := b.profit
	b.bestDataLock.Unlock()

	if bestHeader == nil || bestHeader.ParentHash.String() != parentHashHex {
		respondError(w, http.StatusBadRequest, "unknown payload")
		return
	}

	bid := boostTypes.BuilderBid{
		Header: bestHeader,
		Value:  [32]byte(common.BytesToHash(profit.Bytes())),
		Pubkey: b.builderPublicKey,
	}
	signature, err := boostTypes.SignMessage(&bid, b.builderSigningDomain, b.builderSecretKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := &boostTypes.GetHeaderResponse{
		Version: "bellatrix",
		Data:    &boostTypes.SignedBuilderBid{Message: &bid, Signature: signature},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (b *Backend) handleGetPayload(w http.ResponseWriter, req *http.Request) {
	payload := new(boostTypes.SignedBlindedBeaconBlock)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if len(payload.Signature) != 96 {
		respondError(w, http.StatusBadRequest, "invalid signature")
		return
	}

	nextSlotProposerPubkeyHex, err := b.beaconClient.getProposerForNextSlot(payload.Message.Slot)
	if err != nil {
		if b.enableBeaconChecks {
			respondError(w, http.StatusBadRequest, "unknown validator")
			return
		}
	}

	nextSlotProposerPubkeyBytes, err := hexutil.Decode(string(nextSlotProposerPubkeyHex))
	if err != nil {
		if b.enableBeaconChecks {
			respondError(w, http.StatusBadRequest, "unknown validator")
			return
		}
	}

	ok, err := boostTypes.VerifySignature(payload.Message, b.proposerSigningDomain, nextSlotProposerPubkeyBytes[:], payload.Signature[:])
	if !ok || err != nil {
		if b.enableBeaconChecks {
			respondError(w, http.StatusBadRequest, "invalid signature")
			return
		}
	}

	b.bestDataLock.Lock()
	bestHeader := b.bestHeader
	bestPayload := b.bestPayload
	b.bestDataLock.Unlock()

	if bestHeader == nil || bestPayload == nil {
		respondError(w, http.StatusInternalServerError, "no payloads")
		return
	}

	if !ExecutionPayloadHeaderEqual(bestHeader, payload.Message.Body.ExecutionPayloadHeader) {
		respondError(w, http.StatusBadRequest, "unknown payload")
		return
	}

	response := boostTypes.GetPayloadResponse{
		Version: "bellatrix",
		Data:    bestPayload,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (b *Backend) onForkchoice(payloadAttributes *beacon.PayloadAttributesV1) {
	dataJson, err := json.Marshal(payloadAttributes)
	if err == nil {
		log.Info("FCU", "data", string(dataJson))
	}
	// if payloadAttributes.SuggestedFeeRecipient == common.Address{}
	pubkeyHex, err := b.beaconClient.onForkchoiceUpdate()
	if err != nil {
		return
	}

	if payloadAttributes != nil {
		b.validatorsLock.RLock()
		vd, found := b.validators[pubkeyHex]
		if found {
			payloadAttributes.SuggestedFeeRecipient = [20]byte(vd.FeeRecipient)
			payloadAttributes.GasLimit = vd.GasLimit
		}
		b.validatorsLock.RUnlock()
	}
}

func (b *Backend) newSealedBlock(data *beacon.ExecutableDataV1, block *types.Block) {
	dataJson, err := json.Marshal(data)
	if err == nil {
		log.Info("newSealedBlock", "data", string(dataJson))
	}
	payload := executableDataToExecutionPayload(data)
	payloadHeader, err := payloadToPayloadHeader(payload, data)
	if err != nil {
		log.Error("could not convert payload to header", "err", err)
		return
	}

	b.bestDataLock.Lock()
	b.bestHeader = payloadHeader
	b.bestPayload = payload
	b.profit = new(big.Int).Set(block.Profit)
	b.bestDataLock.Unlock()
}

func payloadToPayloadHeader(p *boostTypes.ExecutionPayload, data *beacon.ExecutableDataV1) (*boostTypes.ExecutionPayloadHeader, error) {
	txs := boostTypes.Transactions{data.Transactions}
	txroot, err := txs.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	return &boostTypes.ExecutionPayloadHeader{
		ParentHash:       p.ParentHash,
		FeeRecipient:     p.FeeRecipient,
		StateRoot:        p.StateRoot,
		ReceiptsRoot:     p.ReceiptsRoot,
		LogsBloom:        p.LogsBloom,
		Random:           p.Random,
		BlockNumber:      p.BlockNumber,
		GasLimit:         p.GasLimit,
		GasUsed:          p.GasUsed,
		Timestamp:        p.Timestamp,
		ExtraData:        data.ExtraData,
		BaseFeePerGas:    p.BaseFeePerGas,
		BlockHash:        p.BlockHash,
		TransactionsRoot: [32]byte(txroot),
	}, nil
}

func executableDataToExecutionPayload(data *beacon.ExecutableDataV1) *boostTypes.ExecutionPayload {
	transactionData := make([]hexutil.Bytes, len(data.Transactions))
	for i, tx := range data.Transactions {
		transactionData[i] = hexutil.Bytes(tx)
	}

	return &boostTypes.ExecutionPayload{
		ParentHash:    [32]byte(data.ParentHash),
		FeeRecipient:  [20]byte(data.FeeRecipient),
		StateRoot:     [32]byte(data.StateRoot),
		ReceiptsRoot:  [32]byte(data.ReceiptsRoot),
		LogsBloom:     boostTypes.Bloom(types.BytesToBloom(data.LogsBloom)),
		Random:        [32]byte(data.Random),
		BlockNumber:   data.Number,
		GasLimit:      data.GasLimit,
		GasUsed:       data.GasUsed,
		Timestamp:     data.Timestamp,
		ExtraData:     data.ExtraData,
		BaseFeePerGas: *new(boostTypes.U256Str).FromBig(data.BaseFeePerGas),
		BlockHash:     [32]byte(data.BlockHash),
		Transactions:  transactionData,
	}
}

func ExecutionPayloadHeaderEqual(l *boostTypes.ExecutionPayloadHeader, r *boostTypes.ExecutionPayloadHeader) bool {
	return l.ParentHash == r.ParentHash && l.FeeRecipient == r.FeeRecipient && l.StateRoot == r.StateRoot && l.ReceiptsRoot == r.ReceiptsRoot && l.LogsBloom == r.LogsBloom && l.Random == r.Random && l.BlockNumber == r.BlockNumber && l.GasLimit == r.GasLimit && l.GasUsed == r.GasUsed && l.Timestamp == r.Timestamp && l.BaseFeePerGas == r.BaseFeePerGas && bytes.Equal(l.ExtraData, r.ExtraData) && l.BlockHash == r.BlockHash && l.TransactionsRoot == r.TransactionsRoot
}
