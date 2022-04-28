package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/flashbots/go-boost-utils/bls"
	boostTypes "github.com/flashbots/go-boost-utils/types"
	"github.com/stretchr/testify/require"
)

func newTestBackend(t *testing.T) (*Backend, *ValidatorPrivateData) {
	validator := NewRandomValidator()
	sk, _ := bls.GenerateRandomSecretKey()
	bDomain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, [4]byte{0x02, 0x0, 0x0, 0x0}, boostTypes.Hash{})
	genesisValidatorsRoot := boostTypes.Hash(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"))
	cDomain := boostTypes.ComputeDomain(boostTypes.DomainTypeBeaconProposer, [4]byte{0x02, 0x0, 0x0, 0x0}, genesisValidatorsRoot)
	backend := NewBackend(sk, &testBeaconClient{validator}, bDomain, cDomain, true)
	// service := NewService("127.0.0.1:31545", backend)

	return backend, validator
}

func testRequest(t *testing.T, backend *Backend, method string, path string, payload any) *httptest.ResponseRecorder {
	var req *http.Request
	var err error

	if payload == nil {
		req, err = http.NewRequest(method, path, nil)
	} else {
		payloadBytes, err2 := json.Marshal(payload)
		require.NoError(t, err2)
		req, err = http.NewRequest(method, path, bytes.NewReader(payloadBytes))
	}

	require.NoError(t, err)
	rr := httptest.NewRecorder()
	getRouter(backend).ServeHTTP(rr, req)
	return rr
}

func TestValidatorRegistration(t *testing.T) {
	backend, _ := newTestBackend(t)
	log.Error("rsk", "sk", hexutil.Encode(backend.builderSecretKey.Serialize()))

	v := NewRandomValidator()
	payload, err := prepareRegistrationMessage(t, backend.builderSigningDomain, v)
	require.NoError(t, err)

	rr := testRequest(t, backend, "POST", "/eth/v1/builder/validators", payload)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, backend.validators, PubkeyHex(v.Pk.String()))
	require.Equal(t, ValidatorData{FeeRecipient: payload[0].Message.FeeRecipient, GasLimit: payload[0].Message.GasLimit, Timestamp: payload[0].Message.Timestamp}, backend.validators[PubkeyHex(v.Pk.String())])

	rr = testRequest(t, backend, "POST", "/eth/v1/builder/validators", payload)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, `{"code":400,"message":"invalid timestamp"}`+"\n", rr.Body.String())

	payload[0].Message.Timestamp += 1
	// Invalid signature
	payload[0].Signature[len(payload[0].Signature)-1] = 0x00
	rr = testRequest(t, backend, "POST", "/eth/v1/builder/validators", payload)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, `{"code":400,"message":"invalid signature"}`+"\n", rr.Body.String())

	// TODO: cover all errors
}

func prepareRegistrationMessage(t *testing.T, domain boostTypes.Domain, v *ValidatorPrivateData) ([]boostTypes.SignedValidatorRegistration, error) {
	var pubkey boostTypes.PublicKey
	pubkey.FromSlice(v.Pk)
	require.Equal(t, []byte(v.Pk), pubkey[:])

	msg := boostTypes.RegisterValidatorRequestMessage{
		FeeRecipient: boostTypes.Address{0x42},
		GasLimit:     15_000_000,
		Timestamp:    uint64(time.Now().Unix()),
		Pubkey:       pubkey,
	}

	signature, err := v.Sign(&msg, domain)
	require.NoError(t, err)

	return []boostTypes.SignedValidatorRegistration{{
		Message:   &msg,
		Signature: signature,
	}}, nil
}

func registerValidator(t *testing.T, v *ValidatorPrivateData, backend *Backend) {
	payload, err := prepareRegistrationMessage(t, backend.builderSigningDomain, v)
	require.NoError(t, err)

	log.Info("Registering", "payload", payload[0].Message)
	rr := testRequest(t, backend, "POST", "/eth/v1/builder/validators", payload)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, backend.validators, PubkeyHex(v.Pk.String()))
	require.Equal(t, ValidatorData{FeeRecipient: payload[0].Message.FeeRecipient, GasLimit: payload[0].Message.GasLimit, Timestamp: payload[0].Message.Timestamp}, backend.validators[PubkeyHex(v.Pk.String())])
}

func TestGetHeader(t *testing.T) {
	backend, validator := newTestBackend(t)

	forkchoiceData := &beacon.ExecutableDataV1{
		ParentHash:    common.HexToHash("0xafafafa"),
		FeeRecipient:  common.Address{0x01},
		BlockHash:     common.HexToHash("0xbfbfbfb"),
		BaseFeePerGas: big.NewInt(12),
		ExtraData:     []byte{},
	}
	forkchoiceBlock := &types.Block{
		Profit: big.NewInt(10),
	}

	path := fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", 0, forkchoiceData.ParentHash.Hex(), validator.Pk.String())
	rr := testRequest(t, backend, "GET", path, nil)
	require.Equal(t, `{"code":400,"message":"unknown validator"}`+"\n", rr.Body.String())

	registerValidator(t, validator, backend)

	rr = testRequest(t, backend, "GET", path, nil)
	require.Equal(t, `{"code":400,"message":"unknown payload"}`+"\n", rr.Body.String())

	path = fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", 0, forkchoiceData.ParentHash.Hex(), NewRandomValidator().Pk.String())
	rr = testRequest(t, backend, "GET", path, nil)
	require.Equal(t, `{"code":400,"message":"unknown validator"}`+"\n", rr.Body.String())

	backend.newSealedBlock(forkchoiceData, forkchoiceBlock)

	path = fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", 0, forkchoiceData.ParentHash.Hex(), validator.Pk.String())
	rr = testRequest(t, backend, "GET", path, nil)
	require.Equal(t, http.StatusOK, rr.Code)

	bid := new(boostTypes.GetHeaderResponse)
	err := json.Unmarshal(rr.Body.Bytes(), bid)
	require.NoError(t, err)

	expectedHeader, err := payloadToPayloadHeader(executableDataToExecutionPayload(forkchoiceData))
	require.NoError(t, err)
	require.EqualValues(t, &boostTypes.BuilderBid{
		Header: expectedHeader,
		Value:  [32]byte(common.BytesToHash(forkchoiceBlock.Profit.Bytes())),
		Pubkey: backend.builderPublicKey,
	}, bid.Data.Message)

	require.Equal(t, forkchoiceData.ParentHash.Bytes(), bid.Data.Message.Header.ParentHash[:], "didn't build on expected parent")
	ok, err := boostTypes.VerifySignature(bid.Data.Message, backend.builderSigningDomain, backend.builderPublicKey[:], bid.Data.Signature[:])

	require.NoError(t, err)
	require.True(t, ok)
}

func TestGetPayload(t *testing.T) {
	backend, validator := newTestBackend(t)

	forkchoiceData := &beacon.ExecutableDataV1{
		ParentHash:    common.HexToHash("0xafafafa"),
		FeeRecipient:  common.Address{0x01},
		BlockHash:     common.HexToHash("0xbfbfbfb"),
		BaseFeePerGas: big.NewInt(12),
		ExtraData:     []byte{},
	}
	forkchoiceBlock := &types.Block{
		Profit: big.NewInt(10),
	}

	registerValidator(t, validator, backend)
	backend.newSealedBlock(forkchoiceData, forkchoiceBlock)

	path := fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", 0, forkchoiceData.ParentHash.Hex(), validator.Pk.String())
	rr := testRequest(t, backend, "GET", path, nil)
	require.Equal(t, http.StatusOK, rr.Code)

	bid := new(boostTypes.GetHeaderResponse)
	err := json.Unmarshal(rr.Body.Bytes(), bid)
	require.NoError(t, err)

	// Create request payload
	msg := &boostTypes.BlindedBeaconBlock{
		Slot:          1,
		ProposerIndex: 2,
		ParentRoot:    boostTypes.Root{0x03},
		StateRoot:     boostTypes.Root{0x04},
		Body: &boostTypes.BlindedBeaconBlockBody{
			Eth1Data: &boostTypes.Eth1Data{
				DepositRoot:  boostTypes.Root{0x05},
				DepositCount: 5,
				BlockHash:    boostTypes.Hash{0x06},
			},
			SyncAggregate: &boostTypes.SyncAggregate{
				CommitteeBits:      boostTypes.CommitteeBits{0x07},
				CommitteeSignature: boostTypes.Signature{0x08},
			},
			ExecutionPayloadHeader: bid.Data.Message.Header,
		},
	}

	// TODO: test wrong signing domain
	signature, err := validator.Sign(msg, backend.proposerSigningDomain)
	require.NoError(t, err)

	// Call getPayload with invalid signature
	rr = testRequest(t, backend, "POST", "/eth/v1/builder/blinded_blocks", boostTypes.SignedBlindedBeaconBlock{
		Message:   msg,
		Signature: boostTypes.Signature{0x09},
	})
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, `{"code":400,"message":"invalid signature"}`+"\n", rr.Body.String())

	// Call getPayload with correct signature
	rr = testRequest(t, backend, "POST", "/eth/v1/builder/blinded_blocks", boostTypes.SignedBlindedBeaconBlock{
		Message:   msg,
		Signature: signature,
	})

	// Verify getPayload response
	require.Equal(t, http.StatusOK, rr.Code)
	getPayloadResponse := new(boostTypes.GetPayloadResponse)
	err = json.Unmarshal(rr.Body.Bytes(), getPayloadResponse)
	require.NoError(t, err)
	require.Equal(t, bid.Data.Message.Header.BlockHash, getPayloadResponse.Data.BlockHash)
}
