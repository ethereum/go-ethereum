// Copyright 2024 The go-ethereum Authors
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
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// SIWEMessage represents a parsed EIP-4361 Sign-In with Ethereum message.
type SIWEMessage struct {
	Scheme         string     // optional
	Domain         string     // required
	Address        string     // required
	Statement      string     // optional
	URI            string     // required
	Version        string     // required
	ChainID        uint64     // required
	Nonce          string     // required
	IssuedAt       time.Time  // required
	ExpirationTime *time.Time // optional
	NotBefore      *time.Time // optional
	RequestID      string     // optional
	Resources      []string   // optional
}

// SIWEWantedPrefix is the phrase that identifies a SIWE message. Wallet
// implementers SHOULD warn users if this appears in any EIP-191 signing
// request that does not fully conform to EIP-4361.
const (
	SIWEWantedPrefix = "wants you to sign in with your Ethereum account"

	siweHeaderSuffix = " wants you to sign in with your Ethereum account:"
)

var (
	siweSchemeRegexp = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+\-.]*$`)
	siweNonceRegexp  = regexp.MustCompile(`^[a-zA-Z0-9]{8,}$`)
)

// validateSIWEMessage inspects a text/plain signing request for EIP-4361 content.
// If the text contains the SIWE phrase it attempts a full parse and domain check,
// returning structured display messages and any validation warnings or errors.
// Returns nil, nil when the text is not SIWE-related.
func validateSIWEMessage(text string, meta Metadata) ([]*apitypes.NameValueType, []apitypes.ValidationInfo) {
	if !strings.Contains(text, SIWEWantedPrefix) {
		return nil, nil
	}
	siweMsg, err := parseSIWEMessage(text)
	if err != nil {
		return nil, []apitypes.ValidationInfo{{
			Typ:     apitypes.WARN,
			Message: fmt.Sprintf("message appears to be Sign-In With Ethereum but is not valid: %v", err),
		}}
	}
	var callInfo []apitypes.ValidationInfo
	switch meta.Scheme {
	case "ipc":
		// no browser involved, no origin to verify
	case "http":
		if meta.Origin == "" {
			callInfo = append(callInfo, apitypes.ValidationInfo{
				Typ:     apitypes.WARN,
				Message: "could not verify domain: request has no Origin header",
			})
		} else {
			origin, err := url.Parse(meta.Origin)
			if err == nil && origin.Host != siweMsg.Domain {
				callInfo = append(callInfo, apitypes.ValidationInfo{
					Typ: apitypes.CRIT,
					Message: fmt.Sprintf("domain mismatch: message claims %q but request origin is %q",
						siweMsg.Domain, origin.Host),
				})
			}
		}
	}
	return siweToAPIMessages(siweMsg), callInfo
}

// parseSIWEMessage parses a Sign-In with Ethereum message as defined by EIP-4361.
// It validates structure, field order, and the format of each field value.
func parseSIWEMessage(msg string) (*SIWEMessage, error) {
	lines := strings.Split(msg, "\n")

	// Minimum: header, address, blank, blank, URI, Version, Chain ID, Nonce, Issued At
	if len(lines) < 9 {
		return nil, errors.New("message too short to be a valid SIWE message")
	}

	cursor := 0
	scheme, domain, err := parseSIWEHeader(lines[cursor])
	if err != nil {
		return nil, err
	}
	cursor++

	if err := validateSIWEAddress(lines[cursor]); err != nil {
		return nil, err
	}
	cursor++

	if lines[cursor] != "" {
		return nil, errors.New("expected empty line after address")
	}
	cursor++

	statement, err := parseSIWEStatement(lines, &cursor)
	if err != nil {
		return nil, err
	}

	uri, version, chainID, nonce, issuedAt, err := parseSIWERequiredFields(lines, &cursor)
	if err != nil {
		return nil, err
	}

	siwe := &SIWEMessage{
		Scheme:    scheme,
		Domain:    domain,
		Address:   lines[1],
		Statement: statement,
		URI:       uri,
		Version:   version,
		ChainID:   chainID,
		Nonce:     nonce,
		IssuedAt:  issuedAt,
	}

	if err := parseSIWEOptionalFields(lines, &cursor, siwe); err != nil {
		return nil, err
	}

	if cursor < len(lines) {
		return nil, fmt.Errorf("unexpected content after SIWE fields: %q", lines[cursor])
	}

	return siwe, nil
}

// parseSIWEHeader extracts the scheme (optional) and domain from line 0.
func parseSIWEHeader(line string) (scheme, domain string, err error) {
	if !strings.HasSuffix(line, siweHeaderSuffix) {
		return "", "", errors.New("first line must end with \" wants you to sign in with your Ethereum account:\"")
	}
	prefix := strings.TrimSuffix(line, siweHeaderSuffix)

	if i := strings.Index(prefix, "://"); i != -1 {
		scheme = prefix[:i]
		domain = prefix[i+3:]
		if !siweSchemeRegexp.MatchString(scheme) {
			return "", "", fmt.Errorf("invalid URI scheme %q", scheme)
		}
	} else {
		domain = prefix
	}

	if err := validateSIWEDomain(domain); err != nil {
		return "", "", err
	}
	return scheme, domain, nil
}

// validateSIWEAddress checks that s is a valid hex Ethereum address with EIP-55 checksum.
func validateSIWEAddress(s string) error {
	if !common.IsHexAddress(s) {
		return errors.New("invalid Ethereum address")
	}
	if common.HexToAddress(s).Hex() != s {
		return errors.New("address does not conform to EIP-55 checksum encoding")
	}
	return nil
}

// parseSIWEStatement reads the optional statement and the blank line that follows it.
// cursor is left pointing at the first key-value field line.
func parseSIWEStatement(lines []string, cursor *int) (string, error) {
	if lines[*cursor] == "" {
		*cursor++
		return "", nil
	}
	statement := lines[*cursor]
	*cursor++
	if *cursor >= len(lines) || lines[*cursor] != "" {
		return "", errors.New("expected empty line after statement")
	}
	*cursor++
	return statement, nil
}

// parseSIWERequiredFields reads URI through Issued At in strict order.
func parseSIWERequiredFields(lines []string, cursor *int) (uri, version string, chainID uint64, nonce string, issuedAt time.Time, err error) {
	uri, err = parseSIWEField(lines, cursor, "URI: ")
	if err != nil {
		return
	}
	if err = validateSIWEURI(uri); err != nil {
		return
	}

	version, err = parseSIWEField(lines, cursor, "Version: ")
	if err != nil {
		return
	}
	if version != "1" {
		err = fmt.Errorf("unsupported SIWE version %q, must be \"1\"", version)
		return
	}

	var chainIDStr string
	chainIDStr, err = parseSIWEField(lines, cursor, "Chain ID: ")
	if err != nil {
		return
	}
	chainID, err = strconv.ParseUint(chainIDStr, 10, 64)
	if err != nil {
		err = fmt.Errorf("invalid Chain ID %q: must be a positive integer", chainIDStr)
		return
	}

	nonce, err = parseSIWEField(lines, cursor, "Nonce: ")
	if err != nil {
		return
	}
	if !siweNonceRegexp.MatchString(nonce) {
		err = errors.New("nonce must be at least 8 alphanumeric characters")
		return
	}

	var issuedAtStr string
	issuedAtStr, err = parseSIWEField(lines, cursor, "Issued At: ")
	if err != nil {
		return
	}
	issuedAt, err = parseSIWEDateTime(issuedAtStr)
	if err != nil {
		err = fmt.Errorf("invalid Issued At: %w", err)
	}
	return
}

// parseSIWEOptionalFields reads Expiration Time, Not Before, Request ID, and
// Resources in strict order. Any unrecognised line is left for the caller to
// detect as unexpected content.
func parseSIWEOptionalFields(lines []string, cursor *int, siwe *SIWEMessage) error {
	if err := parseSIWEOptionalTime(lines, cursor, "Expiration Time: ", &siwe.ExpirationTime); err != nil {
		return err
	}
	if err := parseSIWEOptionalTime(lines, cursor, "Not Before: ", &siwe.NotBefore); err != nil {
		return err
	}
	if *cursor < len(lines) && strings.HasPrefix(lines[*cursor], "Request ID: ") {
		siwe.RequestID = strings.TrimPrefix(lines[*cursor], "Request ID: ")
		(*cursor)++
	}
	return parseSIWEResources(lines, cursor, siwe)
}

// parseSIWEOptionalTime parses an optional datetime field if its prefix is present.
func parseSIWEOptionalTime(lines []string, cursor *int, prefix string, dst **time.Time) error {
	if *cursor >= len(lines) || !strings.HasPrefix(lines[*cursor], prefix) {
		return nil
	}
	val := strings.TrimPrefix(lines[*cursor], prefix)
	t, err := parseSIWEDateTime(val)
	if err != nil {
		return err
	}
	*dst = &t
	(*cursor)++
	return nil
}

// parseSIWEResources reads the Resources section if present.
func parseSIWEResources(lines []string, cursor *int, siwe *SIWEMessage) error {
	if *cursor >= len(lines) || lines[*cursor] != "Resources:" {
		return nil
	}
	(*cursor)++
	for *cursor < len(lines) {
		if !strings.HasPrefix(lines[*cursor], "- ") {
			return fmt.Errorf("invalid resource line %q: must start with \"- \"", lines[*cursor])
		}
		resource := strings.TrimPrefix(lines[*cursor], "- ")
		if err := validateSIWEURI(resource); err != nil {
			return err
		}
		siwe.Resources = append(siwe.Resources, resource)
		(*cursor)++
	}
	return nil
}

// parseSIWEField reads the line at *cursor, strips the expected prefix, advances
// the cursor, and returns the value. Returns an error if the line is missing or
// does not start with prefix.
func parseSIWEField(lines []string, cursor *int, prefix string) (string, error) {
	if *cursor >= len(lines) {
		return "", fmt.Errorf("missing required field %q", strings.TrimRight(prefix, " "))
	}
	if !strings.HasPrefix(lines[*cursor], prefix) {
		return "", fmt.Errorf("expected field %q, got %q", strings.TrimRight(prefix, " "), lines[*cursor])
	}
	val := strings.TrimPrefix(lines[*cursor], prefix)
	(*cursor)++
	return val, nil
}

// parseSIWEDateTime parses an RFC 3339 datetime string, with or without
// sub-second precision.
func parseSIWEDateTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("not a valid RFC 3339 datetime: %q", s)
}

// validateSIWEURI checks that s is a valid RFC 3986 absolute URI.
// url.Parse is too lenient (accepts raw spaces); we check for whitespace
// explicitly since RFC 3986 requires spaces to be percent-encoded.
func validateSIWEURI(s string) error {
	if strings.ContainsAny(s, " \t") {
		return fmt.Errorf("URI %q contains invalid whitespace", s)
	}
	u, err := url.Parse(s)
	if err != nil || !u.IsAbs() {
		return fmt.Errorf("URI %q is not a valid RFC 3986 absolute URI", s)
	}
	return nil
}

// validateSIWEDomain checks that s is a valid RFC 3986 authority (host[:port]).
// Per EIP-4361: domain = authority = [ userinfo "@" ] host [ ":" port ]
func validateSIWEDomain(s string) error {
	if s == "" {
		return errors.New("domain is empty")
	}
	u, err := url.Parse("http://" + s)
	if err != nil {
		return fmt.Errorf("domain %q is not a valid RFC 3986 authority", s)
	}
	// Go splits userinfo into u.User and u.Host, so reconstruct the full
	// authority to verify it round-trips without modification.
	authority := u.Host
	if u.User != nil {
		authority = u.User.String() + "@" + u.Host
	}
	if authority != s {
		return fmt.Errorf("domain %q is not a valid RFC 3986 authority", s)
	}
	return nil
}

func siweToAPIMessages(m *SIWEMessage) []*apitypes.NameValueType {
	nvts := []*apitypes.NameValueType{
		{Name: "Domain", Typ: "domain", Value: m.Domain},
		{Name: "Address", Typ: "address", Value: m.Address},
	}
	if m.Statement != "" {
		nvts = append(nvts, &apitypes.NameValueType{Name: "Statement", Typ: "string", Value: m.Statement})
	}
	nvts = append(nvts,
		&apitypes.NameValueType{Name: "URI", Typ: "uri", Value: m.URI},
		&apitypes.NameValueType{Name: "Version", Typ: "uint", Value: m.Version},
		&apitypes.NameValueType{Name: "Chain ID", Typ: "uint", Value: fmt.Sprintf("%d", m.ChainID)},
		&apitypes.NameValueType{Name: "Nonce", Typ: "string", Value: m.Nonce},
		&apitypes.NameValueType{Name: "Issued At", Typ: "datetime", Value: m.IssuedAt.String()},
	)
	if m.ExpirationTime != nil {
		nvts = append(nvts, &apitypes.NameValueType{Name: "Expiration Time", Typ: "datetime", Value: m.ExpirationTime.String()})
	}
	if m.NotBefore != nil {
		nvts = append(nvts, &apitypes.NameValueType{Name: "Not Before", Typ: "datetime", Value: m.NotBefore.String()})
	}
	if m.RequestID != "" {
		nvts = append(nvts, &apitypes.NameValueType{Name: "Request ID", Typ: "string", Value: m.RequestID})
	}
	if len(m.Resources) > 0 {
		res := make([]*apitypes.NameValueType, len(m.Resources))
		for i, r := range m.Resources {
			res[i] = &apitypes.NameValueType{Name: fmt.Sprintf("%d", i+1), Typ: "uri", Value: r}
		}
		nvts = append(nvts, &apitypes.NameValueType{Name: "Resources", Typ: "list", Value: res})
	}
	return nvts
}
