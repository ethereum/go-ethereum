// Copyright 2025 The go-ethereum Authors
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
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Regular expression to match SIWE messages
// Scheme (optional)
var scheme = `(?:[a-zA-Z][a-zA-Z0-9+\-.]*://)?`

// Domain
var domain = `(?:[^ ]+)`

// Static message
var wantsMsg = ` wants you to sign in with your Ethereum account:\n`

// Ethereum address (with basic 0x prefix + 40 hex digits)
var address = `(?:0x[a-fA-F0-9]{40})\n`

// Optional statement (any line not containing "\n\n")
var statement = `(?:[^\n]+\n)?`

// URI
var uri = `URI: (.+)\n`

// Version
var version = `Version: 1\n`

// Chain ID
var chainID = `Chain ID: (\d+)\n`

// Nonce (min 8 alphanumeric characters)
var nonce = `Nonce: [a-zA-Z0-9]{8,}\n`

// Issued At (RFC 3339 date-time)
var issuedAt = `Issued At: ([0-9T:\-+.Z]+)`

// Optional Expiration Time
var expiration = `(?:\nExpiration Time: ([0-9T:\-+.Z]+))?`

// Optional Not Before
var notBefore = `(?:\nNot Before: ([0-9T:\-+.Z]+))?`

// Optional Request ID
var requestID = `(?:\nRequest ID: ([^\n]+))?`

// Optional Resources
var resources = `(?:\nResources:(?:\n- .+)+)?`

// SIWE Message Regex
var siweMessageRegex = regexp.MustCompile(`(?m)^` + scheme + domain + wantsMsg +
	address + `\n` + statement + `\n` +
	uri + version + chainID + nonce + issuedAt +
	expiration + notBefore + requestID + resources + `$`)

var ErrMalformedSIWEMEssage = errors.New("the message is asking to sign in with Ethereum but does not conform to EIP-4361")

func validateSIWE(req *SignDataRequest) error {
	for _, message := range req.Messages {
		s, ok := message.Value.(string)
		if !ok {
			continue
		}
		if !strings.Contains(s, "wants you to sign in with your Ethereum account") {
			continue
		}
		patterns := siweMessageRegex.FindStringSubmatch(s)
		if len(patterns) != 15 {
			return ErrMalformedSIWEMEssage
		}
		scheme := "https"
		if patterns[0] != "" {
			scheme = patterns[0]
		}
		if err := validateDomain(req, scheme, patterns[1]); err != nil {
			return err
		}
		if err := validateAddress(req, patterns[2]); err != nil {
			return err
		}
	}
	return nil
}

func validateDomain(request *SignDataRequest, scheme, domain string) error {
	siweOrigin := fmt.Sprintf("%s://%s", scheme, domain)
	if siweOrigin != request.Meta.Origin {
		return fmt.Errorf("sign in request domain (%s) does not match source: %s", siweOrigin, request.Meta.Origin)
	}
	return nil
}

func validateAddress(request *SignDataRequest, address string) error {
	checksumAddr := common.HexToAddress(address).Hex()
	requestAddr := request.Address.Address().Hex()

	if checksumAddr != requestAddr {
		return fmt.Errorf("sign in request address (%s) does not match source: %s", checksumAddr, requestAddr)
	}
	return nil
}
