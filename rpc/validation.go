// Copyright 2015 The go-ethereum Authors
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
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// RequestValidator is a HTTP request validator.
// If the request is invalid/rejected, the method returns the http status code
// along with an error.
// If the request is valid/accepted, the method returns (0, nil).
type RequestValidator func(r *http.Request) (int, error)

var DefaultValidators = []RequestValidator{
	ValidateMethod,
	ValidateContentLength,
	ValidateContentType}

// ValidateMethod validates the HTTP method.
func ValidateMethod(r *http.Request) (int, error) {
	if r.Method == http.MethodPut || r.Method == http.MethodDelete {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}
	return 0, nil
}

// ValidateContentLength validates the http Content-Length
func ValidateContentLength(r *http.Request) (int, error) {
	if r.ContentLength > maxRequestContentLength {
		err := fmt.Errorf("content length too large (%d>%d)", r.ContentLength, maxRequestContentLength)
		return http.StatusRequestEntityTooLarge, err
	}
	return 0, nil
}

// ValidateContentType validates the http content type.
func ValidateContentType(r *http.Request) (int, error) {
	// Allow OPTIONS (regardless of content-type)
	if r.Method == http.MethodOptions {
		return 0, nil
	}
	// Check content-type
	if mt, _, err := mime.ParseMediaType(r.Header.Get("content-type")); err == nil {
		for _, accepted := range acceptedContentTypes {
			if accepted == mt {
				return 0, nil
			}
		}
	}
	// Invalid content-type
	err := fmt.Errorf("invalid content type, only %s is supported", contentType)
	return http.StatusUnsupportedMediaType, err
}

// customClaim implements claims.Claim.
type customClaim struct {
	// the `iat` (Issued At) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.6
	IssuedAt int64 `json:"iat,omitempty"`
}

// Valid implements claims.Claim, and checks that the iat is present and valid.
func (c customClaim) Valid() error {
	if time.Now().Unix()-5 < c.IssuedAt {
		return errors.New("token issuance (iat) is too old")
	}
	if time.Now().Unix()+5 > c.IssuedAt {
		return errors.New("token issuance (iat) is too far in the future")
	}
	return nil
}

// MakeJWTValidator creates a validator for jwt tokens.
func MakeJWTValidator(secret []byte) RequestValidator {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	}
	return func(r *http.Request) (int, error) {
		var token string
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
		if len(token) == 0 {
			return http.StatusForbidden, errors.New("missing token")
		}
		var claims customClaim
		t, err := jwt.ParseWithClaims(token, claims, keyFunc, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil {
			return http.StatusForbidden, err
		}
		if !t.Valid {
			// This should not happen, but better safe than sorry if the implementation changes.
			return http.StatusForbidden, errors.New("invalid token")
		}
		return 0, nil
	}
}
