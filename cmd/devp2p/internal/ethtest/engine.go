// Copyright 2023 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/golang-jwt/jwt/v4"
)

// EngineClient is a wrapper around engine-related data.
type EngineClient struct {
	url   string
	jwt   [32]byte
	chain *Chain
	http  *http.Client
}

// NewEngineClient creates a new engine client.
func NewEngineClient(url, jwtSecret string, chain *Chain) *EngineClient {
	return &EngineClient{
		url:   url,
		jwt:   common.HexToHash(jwtSecret),
		chain: chain,
		http:  &http.Client{Timeout: 10 * time.Second},
	}
}

// token returns the jwt claim token for authorization.
func (ec *EngineClient) token() string {
	claims := jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(time.Now())}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(ec.jwt[:])
	return token
}

// rpcRequest marshals a JSON-RPC 2.0 request body for the given method and params.
func rpcRequest(method string, params ...any) ([]byte, error) {
	p, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	return fmt.Appendf(nil, `{"jsonrpc":"2.0","id":1,"method":%q,"params":%s}`, method, p), nil
}

// call sends an Engine API JSON-RPC request and decodes the result into
// result, if non-nil.
func (ec *EngineClient) call(result any, method string, params ...any) error {
	body, err := rpcRequest(method, params...)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, ec.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ec.token())

	resp, err := ec.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: unexpected HTTP status %s", method, resp.Status)
	}
	var reply struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return fmt.Errorf("%s: invalid response: %v", method, err)
	}
	if reply.Error != nil {
		return fmt.Errorf("%s: %s (code %d)", method, reply.Error.Message, reply.Error.Code)
	}
	if result != nil {
		return json.Unmarshal(reply.Result, result)
	}
	return nil
}

// sendForkchoiceUpdated sends an fcu for the head of the generated chain.
func (ec *EngineClient) sendForkchoiceUpdated(custody *types.CustodyBitmap) error {
	head := ec.chain.Head().Hash()
	state := engine.ForkchoiceStateV1{
		HeadBlockHash:      head,
		SafeBlockHash:      head,
		FinalizedBlockHash: head,
	}
	var (
		resp engine.ForkChoiceResponse
		err  error
	)
	if custody != nil {
		err = ec.call(&resp, "engine_forkchoiceUpdatedV4", state, nil, custody)
	} else {
		err = ec.call(&resp, "engine_forkchoiceUpdatedV3", state, nil)
	}
	if err != nil {
		return err
	}
	if status := resp.PayloadStatus; status.Status != engine.VALID {
		msg := ""
		if status.ValidationError != nil {
			msg = ": " + *status.ValidationError
		}
		return fmt.Errorf("forkchoice update not accepted: status %s%s", status.Status, msg)
	}
	return nil
}
