// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"context"
	"errors"
	"fmt"
	"mime"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// sign receives a request and produces a signature
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons, if legacyV==true.
func (api *SignerAPI) sign(req *SignDataRequest, legacyV bool) (hexutil.Bytes, error) {
	// We make the request prior to looking up if we actually have the account, to prevent
	// account-enumeration via the API
	res, err := api.UI.ApproveSignData(req)
	if err != nil {
		return nil, err
	}
	if !res.Approved {
		return nil, ErrRequestDenied
	}
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: req.Address.Address()}
	wallet, err := api.am.Find(account)
	if err != nil {
		return nil, err
	}
	pw, err := api.lookupOrQueryPassword(account.Address,
		"Password for signing",
		fmt.Sprintf("Please enter password for signing data with account %s", account.Address.Hex()))
	if err != nil {
		return nil, err
	}
	// Sign the data with the wallet
	signature, err := wallet.SignDataWithPassphrase(account, pw, req.ContentType, req.Rawdata)
	if err != nil {
		return nil, err
	}
	if legacyV {
		signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}
	return signature, nil
}

// SignData signs the hash of the provided data, but does so differently
// depending on the content-type specified.
//
// Different types of validation occur.
func (api *SignerAPI) SignData(ctx context.Context, contentType string, addr common.MixedcaseAddress, data interface{}) (hexutil.Bytes, error) {
	var req, transformV, err = api.determineSignatureFormat(ctx, contentType, addr, data)
	if err != nil {
		return nil, err
	}
	signature, err := api.sign(req, transformV)
	if err != nil {
		api.UI.ShowError(err.Error())
		return nil, err
	}
	return signature, nil
}

// determineSignatureFormat determines which signature method should be used based upon the mime type
// In the cases where it matters ensure that the charset is handled. The charset
// resides in the 'params' returned as the second returnvalue from mime.ParseMediaType
// charset, ok := params["charset"]
// As it is now, we accept any charset and just treat it as 'raw'.
// This method returns the mimetype for signing along with the request
func (api *SignerAPI) determineSignatureFormat(ctx context.Context, contentType string, addr common.MixedcaseAddress, data interface{}) (*SignDataRequest, bool, error) {
	var (
		req          *SignDataRequest
		useEthereumV = true // Default to use V = 27 or 28, the legacy Ethereum format
	)
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, useEthereumV, err
	}

	switch mediaType {
	case apitypes.IntendedValidator.Mime:
		// Data with an intended validator
		validatorData, err := UnmarshalValidatorData(data)
		if err != nil {
			return nil, useEthereumV, err
		}
		sighash, msg := SignTextValidator(validatorData)
		messages := []*apitypes.NameValueType{
			{
				Name:  "This is a request to sign data intended for a particular validator (see EIP 191 version 0)",
				Typ:   "description",
				Value: "",
			},
			{
				Name:  "Intended validator address",
				Typ:   "address",
				Value: validatorData.Address.String(),
			},
			{
				Name:  "Application-specific data",
				Typ:   "hexdata",
				Value: validatorData.Message,
			},
			{
				Name:  "Full message for signing",
				Typ:   "hexdata",
				Value: fmt.Sprintf("0x%x", msg),
			},
		}
		req = &SignDataRequest{ContentType: mediaType, Rawdata: []byte(msg), Messages: messages, Hash: sighash}
	case apitypes.ApplicationClique.Mime:
		// Clique is the Ethereum PoA standard
		stringData, ok := data.(string)
		if !ok {
			return nil, useEthereumV, fmt.Errorf("input for %v must be an hex-encoded string", apitypes.ApplicationClique.Mime)
		}
		cliqueData, err := hexutil.Decode(stringData)
		if err != nil {
			return nil, useEthereumV, err
		}
		header := &types.Header{}
		if err := rlp.DecodeBytes(cliqueData, header); err != nil {
			return nil, useEthereumV, err
		}
		// The incoming clique header is already truncated, sent to us with a extradata already shortened
		if len(header.Extra) < 65 {
			// Need to add it back, to get a suitable length for hashing
			newExtra := make([]byte, len(header.Extra)+65)
			copy(newExtra, header.Extra)
			header.Extra = newExtra
		}
		// Get back the rlp data, encoded by us
		sighash, cliqueRlp, err := cliqueHeaderHashAndRlp(header)
		if err != nil {
			return nil, useEthereumV, err
		}
		messages := []*apitypes.NameValueType{
			{
				Name:  "Clique header",
				Typ:   "clique",
				Value: fmt.Sprintf("clique header %d [0x%x]", header.Number, header.Hash()),
			},
		}
		// Clique uses V on the form 0 or 1
		useEthereumV = false
		req = &SignDataRequest{ContentType: mediaType, Rawdata: cliqueRlp, Messages: messages, Hash: sighash}
	default: // also case TextPlain.Mime:
		// Calculates an Ethereum ECDSA signature for:
		// hash = keccak256("\x19${byteVersion}Ethereum Signed Message:\n${message length}${message}")
		// We expect it to be a string
		if stringData, ok := data.(string); !ok {
			return nil, useEthereumV, fmt.Errorf("input for text/plain must be an hex-encoded string")
		} else {
			if textData, err := hexutil.Decode(stringData); err != nil {
				return nil, useEthereumV, err
			} else {
				sighash, msg := accounts.TextAndHash(textData)
				messages := []*apitypes.NameValueType{
					{
						Name:  "message",
						Typ:   accounts.MimetypeTextPlain,
						Value: msg,
					},
				}
				req = &SignDataRequest{ContentType: mediaType, Rawdata: []byte(msg), Messages: messages, Hash: sighash}
			}
		}
	}
	req.Address = addr
	req.Meta = MetadataFromContext(ctx)
	return req, useEthereumV, nil
}

// SignTextWithValidator signs the given message which can be further recovered
// with the given validator.
// hash = keccak256("\x19\x00"${address}${data}).
func SignTextValidator(validatorData apitypes.ValidatorData) (hexutil.Bytes, string) {
	msg := fmt.Sprintf("\x19\x00%s%s", string(validatorData.Address.Bytes()), string(validatorData.Message))
	return crypto.Keccak256([]byte(msg)), msg
}

// cliqueHeaderHashAndRlp returns the hash which is used as input for the proof-of-authority
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// The method requires the extra data to be at least 65 bytes -- the original implementation
// in clique.go panics if this is the case, thus it's been reimplemented here to avoid the panic
// and simply return an error instead
func cliqueHeaderHashAndRlp(header *types.Header) (hash, rlp []byte, err error) {
	if len(header.Extra) < 65 {
		err = fmt.Errorf("clique header extradata too short, %d < 65", len(header.Extra))
		return
	}
	rlp = clique.CliqueRLP(header)
	hash = clique.SealHash(header).Bytes()
	return hash, rlp, err
}

// SignTypedData signs EIP-712 conformant typed data
// hash = keccak256("\x19${byteVersion}${domainSeparator}${hashStruct(message)}")
// It returns
// - the signature,
// - and/or any error
func (api *SignerAPI) SignTypedData(ctx context.Context, addr common.MixedcaseAddress, typedData apitypes.TypedData) (hexutil.Bytes, error) {
	signature, _, err := api.signTypedData(ctx, addr, typedData, nil)
	return signature, err
}

// signTypedData is identical to the capitalized version, except that it also returns the hash (preimage)
// - the signature preimage (hash)
func (api *SignerAPI) signTypedData(ctx context.Context, addr common.MixedcaseAddress,
	typedData apitypes.TypedData, validationMessages *apitypes.ValidationMessages) (hexutil.Bytes, hexutil.Bytes, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, nil, err
	}
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, nil, err
	}
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	sighash := crypto.Keccak256(rawData)
	messages, err := typedData.Format()
	if err != nil {
		return nil, nil, err
	}
	req := &SignDataRequest{
		ContentType: apitypes.DataTyped.Mime,
		Rawdata:     rawData,
		Messages:    messages,
		Hash:        sighash,
		Address:     addr}
	if validationMessages != nil {
		req.Callinfo = validationMessages.Messages
	}
	signature, err := api.sign(req, true)
	if err != nil {
		api.UI.ShowError(err.Error())
		return nil, nil, err
	}
	return signature, sighash, nil
}

// EcRecover recovers the address associated with the given sig.
// Only compatible with `text/plain`
func (api *SignerAPI) EcRecover(ctx context.Context, data hexutil.Bytes, sig hexutil.Bytes) (common.Address, error) {
	// Returns the address for the Account that was used to create the signature.
	//
	// Note, this function is compatible with eth_sign and personal_sign. As such it recovers
	// the address of:
	// hash = keccak256("\x19${byteVersion}Ethereum Signed Message:\n${message length}${message}")
	// addr = ecrecover(hash, signature)
	//
	// Note, the signature must conform to the secp256k1 curve R, S and V values, where
	// the V value must be be 27 or 28 for legacy reasons.
	//
	// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_ecRecover
	if len(sig) != 65 {
		return common.Address{}, fmt.Errorf("signature must be 65 bytes long")
	}
	if sig[64] != 27 && sig[64] != 28 {
		return common.Address{}, fmt.Errorf("invalid Ethereum signature (V is not 27 or 28)")
	}
	sig[64] -= 27 // Transform yellow paper V from 27/28 to 0/1
	hash := accounts.TextHash(data)
	rpk, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*rpk), nil
}

// UnmarshalValidatorData converts the bytes input to typed data
func UnmarshalValidatorData(data interface{}) (apitypes.ValidatorData, error) {
	raw, ok := data.(map[string]interface{})
	if !ok {
		return apitypes.ValidatorData{}, errors.New("validator input is not a map[string]interface{}")
	}
	addr, ok := raw["address"].(string)
	if !ok {
		return apitypes.ValidatorData{}, errors.New("validator address is not sent as a string")
	}
	addrBytes, err := hexutil.Decode(addr)
	if err != nil {
		return apitypes.ValidatorData{}, err
	}
	if !ok || len(addrBytes) == 0 {
		return apitypes.ValidatorData{}, errors.New("validator address is undefined")
	}

	message, ok := raw["message"].(string)
	if !ok {
		return apitypes.ValidatorData{}, errors.New("message is not sent as a string")
	}
	messageBytes, err := hexutil.Decode(message)
	if err != nil {
		return apitypes.ValidatorData{}, err
	}
	if !ok || len(messageBytes) == 0 {
		return apitypes.ValidatorData{}, errors.New("message is undefined")
	}

	return apitypes.ValidatorData{
		Address: common.BytesToAddress(addrBytes),
		Message: messageBytes,
	}, nil
}
