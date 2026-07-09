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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang-jwt/jwt/v4"
)

// EngineClient is a wrapper around engine-related data.
type EngineClient struct {
	url     string
	jwt     [32]byte
	headfcu []byte
}

// NewEngineClient creates a new engine client.
func NewEngineClient(dir, url, jwt string) (*EngineClient, error) {
	headfcu, err := os.ReadFile(filepath.Join(dir, "headfcu.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read headfcu: %w", err)
	}
	return &EngineClient{url, common.HexToHash(jwt), headfcu}, nil
}

// token returns the jwt claim token for authorization.
func (ec *EngineClient) token() string {
	claims := jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(time.Now())}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(ec.jwt[:])
	return token
}

// sendForkchoiceUpdated sends an fcu for the head of the generated chain.
func (ec *EngineClient) sendForkchoiceUpdated() error {
	var (
		req, _ = http.NewRequest(http.MethodPost, ec.url, io.NopCloser(bytes.NewReader(ec.headfcu)))
		header = make(http.Header)
	)
	// Set header
	header.Set("accept", "application/json")
	header.Set("content-type", "application/json")
	header.Set("Authorization", fmt.Sprintf("Bearer %v", ec.token()))
	req.Header = header

	_, err := new(http.Client).Do(req)
	return err
}
