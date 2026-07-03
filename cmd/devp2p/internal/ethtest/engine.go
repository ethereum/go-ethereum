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

// call sends an authenticated Engine API JSON-RPC request. Response body is
// not inspected — only transport errors are returned.
func (ec *EngineClient) call(method string, params ...any) error {
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

	_, err = ec.http.Do(req)
	return err
}

// sendForkchoiceUpdated sends an fcu for the head of the generated chain.
func (ec *EngineClient) sendForkchoiceUpdated() error {
	head := ec.chain.Head().Hash()
	state := engine.ForkchoiceStateV1{
		HeadBlockHash:      head,
		SafeBlockHash:      head,
		FinalizedBlockHash: head,
	}
	return ec.call("engine_forkchoiceUpdatedV3", state, nil)
}
