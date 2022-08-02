// Copyright 2022 The go-ethereum Authors
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

package rpc

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// HeaderAuthProvider is an interface for adding JWT Bearer Tokens to HTTP/WS (on the initial upgrade)
// requests to authenticated APIs.
// See https://github.com/ethereum/execution-apis/blob/main/src/engine/authentication.md for details
// about the authentication scheme.
type HeaderAuthProvider interface {
	// AddAuthHeader adds an up to date Authorization Bearer token field to the header
	AddAuthHeader(header *http.Header) error
}

type JWTAuthProvider struct {
	secret [32]byte
}

// NewJWTAuthProvider creates a new JWT Auth Provider.
// The secret MUST be 32 bytes (256 bits) as defined by the Engine-API authentication spec.
func NewJWTAuthProvider(jwtsecret [32]byte) *JWTAuthProvider {
	return &JWTAuthProvider{secret: jwtsecret}
}

// AddAuthHeader adds a JWT Authorization token to the header
func (p *JWTAuthProvider) AddAuthHeader(header *http.Header) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": &jwt.NumericDate{Time: time.Now()},
	})
	s, err := token.SignedString(p.secret[:])
	if err != nil {
		return fmt.Errorf("failed to create JWT token: %w", err)
	}
	header.Add("Authorization", "Bearer "+s)
	return nil
}
