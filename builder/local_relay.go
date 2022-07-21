package builder

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/flashbots/go-boost-utils/bls"
	boostTypes "github.com/flashbots/go-boost-utils/types"
	"github.com/gorilla/mux"
)

type ForkData struct {
	GenesisForkVersion    string
	BellatrixForkVersion  string
	GenesisValidatorsRoot string
}

type LocalRelay struct {
	beaconClient IBeaconClient

	relaySecretKey        *bls.SecretKey
	relayPublicKey        boostTypes.PublicKey
	serializedRelayPubkey hexutil.Bytes

	builderSigningDomain  boostTypes.Domain
	proposerSigningDomain boostTypes.Domain

	validatorsLock sync.RWMutex
	validators     map[PubkeyHex]ValidatorData

	enableBeaconChecks bool

	bestDataLock sync.Mutex
	bestHeader   *boostTypes.ExecutionPayloadHeader
	bestPayload  *boostTypes.ExecutionPayload
	profit       boostTypes.U256Str

	indexTemplate *template.Template
	fd            ForkData
}

func NewLocalRelay(sk *bls.SecretKey, beaconClient IBeaconClient, builderSigningDomain boostTypes.Domain, proposerSigningDomain boostTypes.Domain, fd ForkData, enableBeaconChecks bool) *LocalRelay {
	pkBytes := bls.PublicKeyFromSecretKey(sk).Compress()
	pk := boostTypes.PublicKey{}
	pk.FromSlice(pkBytes)

	indexTemplate, err := parseIndexTemplate()
	if err != nil {
		log.Error("could not parse index template", "err", err)
		indexTemplate = nil
	}

	return &LocalRelay{
		beaconClient: beaconClient,

		relaySecretKey: sk,
		relayPublicKey: pk,

		builderSigningDomain:  builderSigningDomain,
		proposerSigningDomain: proposerSigningDomain,
		serializedRelayPubkey: pkBytes,

		validators: make(map[PubkeyHex]ValidatorData),

		enableBeaconChecks: enableBeaconChecks,

		indexTemplate: indexTemplate,
		fd:            fd,
	}
}

func (r *LocalRelay) SubmitBlock(msg *boostTypes.BuilderSubmitBlockRequest) error {
	payloadHeader, err := boostTypes.PayloadToPayloadHeader(msg.ExecutionPayload)
	if err != nil {
		log.Error("could not convert payload to header", "err", err)
		return err
	}

	r.bestDataLock.Lock()
	r.bestHeader = payloadHeader
	r.bestPayload = msg.ExecutionPayload
	r.profit = msg.Message.Value
	r.bestDataLock.Unlock()

	return nil
}

func (r *LocalRelay) handleRegisterValidator(w http.ResponseWriter, req *http.Request) {
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

		ok, err := boostTypes.VerifySignature(registerRequest.Message, r.builderSigningDomain, registerRequest.Message.Pubkey[:], registerRequest.Signature[:])
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
		if !r.beaconClient.isValidator(pubkeyHex) {
			respondError(w, http.StatusBadRequest, "not a validator")
			return
		}
	}

	r.validatorsLock.Lock()
	defer r.validatorsLock.Unlock()

	for _, registerRequest := range payload {
		pubkeyHex := PubkeyHex(registerRequest.Message.Pubkey.String())
		if previousValidatorData, ok := r.validators[pubkeyHex]; ok {
			if registerRequest.Message.Timestamp < previousValidatorData.Timestamp {
				respondError(w, http.StatusBadRequest, "invalid timestamp")
				return
			}

			if registerRequest.Message.Timestamp == previousValidatorData.Timestamp && (registerRequest.Message.FeeRecipient != previousValidatorData.FeeRecipient || registerRequest.Message.GasLimit != previousValidatorData.GasLimit) {
				respondError(w, http.StatusBadRequest, "invalid timestamp")
				return
			}
		}
	}

	for _, registerRequest := range payload {
		pubkeyHex := PubkeyHex(strings.ToLower(registerRequest.Message.Pubkey.String()))
		r.validators[pubkeyHex] = ValidatorData{
			Pubkey:       pubkeyHex,
			FeeRecipient: registerRequest.Message.FeeRecipient,
			GasLimit:     registerRequest.Message.GasLimit,
			Timestamp:    registerRequest.Message.Timestamp,
		}

		log.Info("registered validator", "pubkey", pubkeyHex, "data", r.validators[pubkeyHex])
	}

	w.WriteHeader(http.StatusOK)
}

func (r *LocalRelay) GetValidatorForSlot(nextSlot uint64) (ValidatorData, error) {
	pubkeyHex, err := r.beaconClient.getProposerForNextSlot(nextSlot)
	if err != nil {
		return ValidatorData{}, err
	}

	r.validatorsLock.RLock()
	if vd, ok := r.validators[pubkeyHex]; ok {
		r.validatorsLock.RUnlock()
		return vd, nil
	}
	r.validatorsLock.RUnlock()
	log.Info("no local entry for validator", "validator", pubkeyHex)
	return ValidatorData{}, errors.New("missing validator")
}

func (r *LocalRelay) handleGetHeader(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	slot, err := strconv.Atoi(vars["slot"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "incorrect slot")
		return
	}
	parentHashHex := vars["parent_hash"]
	pubkeyHex := PubkeyHex(strings.ToLower(vars["pubkey"]))

	// Do not validate slot separately, it will create a race between slot update and proposer key
	if nextSlotProposer, err := r.beaconClient.getProposerForNextSlot(uint64(slot)); err != nil || nextSlotProposer != pubkeyHex {
		log.Error("getHeader requested for public key other than next slots proposer", "requested", pubkeyHex, "expected", nextSlotProposer)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Only check if slot is within a couple of the expected one, otherwise will force validators resync
	vd, err := r.GetValidatorForSlot(uint64(slot))
	if err != nil {
		respondError(w, http.StatusBadRequest, "unknown validator")
		return
	}
	if vd.Pubkey != pubkeyHex {
		respondError(w, http.StatusBadRequest, "unknown validator")
		return
	}

	r.bestDataLock.Lock()
	bestHeader := r.bestHeader
	profit := r.profit
	r.bestDataLock.Unlock()

	if bestHeader == nil || bestHeader.ParentHash.String() != parentHashHex {
		respondError(w, http.StatusBadRequest, "unknown payload")
		return
	}

	bid := boostTypes.BuilderBid{
		Header: bestHeader,
		Value:  profit,
		Pubkey: r.relayPublicKey,
	}
	signature, err := boostTypes.SignMessage(&bid, r.builderSigningDomain, r.relaySecretKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := &boostTypes.GetHeaderResponse{
		Version: "bellatrix",
		Data:    &boostTypes.SignedBuilderBid{Message: &bid, Signature: signature},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (r *LocalRelay) handleGetPayload(w http.ResponseWriter, req *http.Request) {
	payload := new(boostTypes.SignedBlindedBeaconBlock)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if len(payload.Signature) != 96 {
		respondError(w, http.StatusBadRequest, "invalid signature")
		return
	}

	nextSlotProposerPubkeyHex, err := r.beaconClient.getProposerForNextSlot(payload.Message.Slot)
	if err != nil {
		if r.enableBeaconChecks {
			respondError(w, http.StatusBadRequest, "unknown validator")
			return
		}
	}

	nextSlotProposerPubkeyBytes, err := hexutil.Decode(string(nextSlotProposerPubkeyHex))
	if err != nil {
		if r.enableBeaconChecks {
			respondError(w, http.StatusBadRequest, "unknown validator")
			return
		}
	}

	ok, err := boostTypes.VerifySignature(payload.Message, r.proposerSigningDomain, nextSlotProposerPubkeyBytes[:], payload.Signature[:])
	if !ok || err != nil {
		if r.enableBeaconChecks {
			respondError(w, http.StatusBadRequest, "invalid signature")
			return
		}
	}

	r.bestDataLock.Lock()
	bestHeader := r.bestHeader
	bestPayload := r.bestPayload
	r.bestDataLock.Unlock()

	log.Info("Received blinded block", "payload", payload, "bestHeader", bestHeader)

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
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
}

func (r *LocalRelay) handleIndex(w http.ResponseWriter, req *http.Request) {
	if r.indexTemplate == nil {
		http.Error(w, "not available", http.StatusInternalServerError)
	}

	r.validatorsLock.RLock()
	noValidators := len(r.validators)
	r.validatorsLock.RUnlock()
	validatorsStats := fmt.Sprint(noValidators) + " validators registered"

	r.bestDataLock.Lock()
	header := r.bestHeader
	payload := r.bestPayload
	r.bestDataLock.Lock()

	headerData, err := json.MarshalIndent(header, "", "  ")
	if err != nil {
		headerData = []byte{}
	}

	payloadData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		payloadData = []byte{}
	}

	statusData := struct {
		Pubkey                string
		ValidatorsStats       string
		GenesisForkVersion    string
		BellatrixForkVersion  string
		GenesisValidatorsRoot string
		BuilderSigningDomain  string
		ProposerSigningDomain string
		Header                string
		Blocks                string
	}{hexutil.Encode(r.serializedRelayPubkey), validatorsStats, r.fd.GenesisForkVersion, r.fd.BellatrixForkVersion, r.fd.GenesisValidatorsRoot, hexutil.Encode(r.builderSigningDomain[:]), hexutil.Encode(r.proposerSigningDomain[:]), string(headerData), string(payloadData)}

	if err := r.indexTemplate.Execute(w, statusData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

func (r *LocalRelay) handleStatus(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func ExecutionPayloadHeaderEqual(l *boostTypes.ExecutionPayloadHeader, r *boostTypes.ExecutionPayloadHeader) bool {
	return l.ParentHash == r.ParentHash && l.FeeRecipient == r.FeeRecipient && l.StateRoot == r.StateRoot && l.ReceiptsRoot == r.ReceiptsRoot && l.LogsBloom == r.LogsBloom && l.Random == r.Random && l.BlockNumber == r.BlockNumber && l.GasLimit == r.GasLimit && l.GasUsed == r.GasUsed && l.Timestamp == r.Timestamp && l.BaseFeePerGas == r.BaseFeePerGas && bytes.Equal(l.ExtraData, r.ExtraData) && l.BlockHash == r.BlockHash && l.TransactionsRoot == r.TransactionsRoot
}
