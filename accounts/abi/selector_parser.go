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

package abi

import (
	"errors"
	"fmt"
	"strings"
)

type SelectorMarshaling struct {
	Name            string               `json:"name"`
	Type            string               `json:"type"`
	Inputs          []ArgumentMarshaling `json:"inputs"`
	Outputs         []ArgumentMarshaling `json:"outputs,omitempty"`
	StateMutability string               `json:"stateMutability,omitempty"`
	Anonymous       bool                 `json:"anonymous,omitempty"`
}

type EventMarshaling struct {
	Name      string               `json:"name"`
	Type      string               `json:"type"`
	Inputs    []ArgumentMarshaling `json:"inputs"`
	Anonymous bool                 `json:"anonymous"`
}

type ErrorMarshaling struct {
	Name   string               `json:"name"`
	Type   string               `json:"type"`
	Inputs []ArgumentMarshaling `json:"inputs"`
}

// ABIMarshaling is a union type that can represent any ABI element
type ABIMarshaling map[string]interface{}

// ParseHumanReadableABI parses a human-readable ABI signature into a JSON-compatible map
func ParseHumanReadableABI(signature string) (ABIMarshaling, error) {
	signature = skipWhitespace(signature)

	if strings.HasPrefix(signature, "constructor") {
		return ParseConstructor(signature)
	}

	if strings.HasPrefix(signature, "fallback") {
		return ParseFallback(signature)
	}

	if strings.HasPrefix(signature, "receive") {
		return ParseReceive(signature)
	}

	if strings.HasPrefix(signature, "event ") || (strings.Contains(signature, "(") && strings.Contains(signature, "indexed")) {
		event, err := ParseEvent(signature)
		if err != nil {
			return nil, err
		}
		result := make(ABIMarshaling)
		result["name"] = event.Name
		result["type"] = event.Type
		result["inputs"] = event.Inputs
		result["anonymous"] = event.Anonymous
		return result, nil
	}

	if strings.HasPrefix(signature, "error ") {
		errSig, err := ParseError(signature)
		if err != nil {
			return nil, err
		}
		result := make(ABIMarshaling)
		result["name"] = errSig.Name
		result["type"] = errSig.Type
		result["inputs"] = errSig.Inputs
		return result, nil
	}

	if strings.HasPrefix(signature, "struct ") {
		return nil, fmt.Errorf("struct definitions not supported, use inline tuple syntax")
	}

	fn, err := ParseSelector(signature)
	if err != nil {
		return nil, err
	}
	result := make(ABIMarshaling)
	result["name"] = fn.Name
	result["type"] = fn.Type
	result["inputs"] = fn.Inputs
	if len(fn.Outputs) > 0 {
		result["outputs"] = fn.Outputs
	}
	result["stateMutability"] = fn.StateMutability
	return result, nil
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentifierSymbol(c byte) bool {
	return c == '$' || c == '_'
}

func parseToken(unescapedSelector string, isIdent bool) (string, string, error) {
	if len(unescapedSelector) == 0 {
		return "", "", errors.New("empty token")
	}
	firstChar := unescapedSelector[0]
	position := 1
	if !(isAlpha(firstChar) || (isIdent && isIdentifierSymbol(firstChar))) {
		return "", "", fmt.Errorf("invalid token start: %c", firstChar)
	}
	for position < len(unescapedSelector) {
		char := unescapedSelector[position]
		if !(isAlpha(char) || isDigit(char) || (isIdent && isIdentifierSymbol(char))) {
			break
		}
		position++
	}
	return unescapedSelector[:position], unescapedSelector[position:], nil
}

func parseIdentifier(unescapedSelector string) (string, string, error) {
	return parseToken(unescapedSelector, true)
}

func skipWhitespace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	return s[i:]
}

// parseKeyword checks if the string starts with a keyword followed by whitespace or special char
func parseKeyword(s string, keyword string) (string, bool) {
	s = skipWhitespace(s)
	if !strings.HasPrefix(s, keyword) {
		return s, false
	}
	rest := s[len(keyword):]
	if len(rest) > 0 && (isAlpha(rest[0]) || isDigit(rest[0]) || isIdentifierSymbol(rest[0])) {
		return s, false
	}
	return skipWhitespace(rest), true
}

// normalizeType normalizes bare int/uint to int256/uint256, including arrays
func normalizeType(typeName string) string {
	if typeName == "int" {
		return "int256"
	}
	if typeName == "uint" {
		return "uint256"
	}
	if strings.HasPrefix(typeName, "int[") {
		return "int256" + typeName[3:]
	}
	if strings.HasPrefix(typeName, "uint[") {
		return "uint256" + typeName[4:]
	}
	return typeName
}

func parseElementaryType(unescapedSelector string) (string, string, error) {
	parsedType, rest, err := parseToken(unescapedSelector, false)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse elementary type: %v", err)
	}
	for len(rest) > 0 && rest[0] == '[' {
		parsedType = parsedType + string(rest[0])
		rest = rest[1:]
		for len(rest) > 0 && isDigit(rest[0]) {
			parsedType = parsedType + string(rest[0])
			rest = rest[1:]
		}
		if len(rest) == 0 || rest[0] != ']' {
			return "", "", fmt.Errorf("failed to parse array: expected ']', got %c", unescapedSelector[0])
		}
		parsedType = parsedType + string(rest[0])
		rest = rest[1:]
	}
	parsedType = normalizeType(parsedType)
	return parsedType, rest, nil
}

// parseTupleType parses inline tuple syntax like (string denom, uint256 amount)[]
func parseTupleType(params string) ([]ArgumentMarshaling, string, string, error) {
	if params == "" || params[0] != '(' {
		return nil, "", params, fmt.Errorf("expected '(' at start of tuple")
	}

	rest := params[1:]
	rest = skipWhitespace(rest)

	if rest[0] == ')' {
		rest = rest[1:]
		arraySuffix := ""
		for len(rest) > 0 && rest[0] == '[' {
			endBracket := 1
			for endBracket < len(rest) && rest[endBracket] != ']' {
				endBracket++
			}
			if endBracket >= len(rest) {
				return nil, "", rest, fmt.Errorf("unclosed array bracket")
			}
			arraySuffix += rest[:endBracket+1]
			rest = rest[endBracket+1:]
		}
		return []ArgumentMarshaling{}, arraySuffix, rest, nil
	}

	var components []ArgumentMarshaling
	paramIndex := 0

	for {
		rest = skipWhitespace(rest)

		var component ArgumentMarshaling
		var err error

		if rest[0] == '(' {
			subComponents, subArraySuffix, newRest, subErr := parseTupleType(rest)
			if subErr != nil {
				return nil, "", rest, fmt.Errorf("failed to parse nested tuple: %v", subErr)
			}
			rest = newRest
			rest = skipWhitespace(rest)

			paramName := fmt.Sprintf("param%d", paramIndex)
			if len(rest) > 0 && (isAlpha(rest[0]) || isIdentifierSymbol(rest[0])) {
				paramName, rest, err = parseIdentifier(rest)
				if err != nil {
					return nil, "", rest, err
				}
			}

			component = ArgumentMarshaling{
				Name:       paramName,
				Type:       "tuple" + subArraySuffix,
				Components: subComponents,
			}
		} else {
			var typeName string
			typeName, rest, err = parseElementaryType(rest)
			if err != nil {
				return nil, "", rest, fmt.Errorf("failed to parse type: %v", err)
			}

			rest = skipWhitespace(rest)

			paramName := fmt.Sprintf("param%d", paramIndex)
			if len(rest) > 0 && (isAlpha(rest[0]) || isIdentifierSymbol(rest[0])) {
				paramName, rest, err = parseIdentifier(rest)
				if err != nil {
					return nil, "", rest, err
				}
			}

			component = ArgumentMarshaling{
				Name: paramName,
				Type: typeName,
			}
		}

		components = append(components, component)
		rest = skipWhitespace(rest)

		if rest[0] == ')' {
			rest = rest[1:]
			arraySuffix := ""
			for len(rest) > 0 && rest[0] == '[' {
				endBracket := 1
				for endBracket < len(rest) && rest[endBracket] != ']' {
					endBracket++
				}
				if endBracket >= len(rest) {
					return nil, "", rest, fmt.Errorf("unclosed array bracket")
				}
				arraySuffix += rest[:endBracket+1]
				rest = rest[endBracket+1:]
			}
			return components, arraySuffix, rest, nil
		} else if rest[0] == ',' {
			rest = rest[1:]
			paramIndex++
		} else {
			return nil, "", rest, fmt.Errorf("expected ',' or ')' in tuple, got '%c'", rest[0])
		}
	}
}

// parseParameterList parses a parameter list with optional indexed flags
func parseParameterList(params string, allowIndexed bool) ([]ArgumentMarshaling, error) {
	params = skipWhitespace(params)
	if params == "" {
		return []ArgumentMarshaling{}, nil
	}

	var arguments []ArgumentMarshaling
	paramIndex := 0

	for params != "" {
		params = skipWhitespace(params)

		var arg ArgumentMarshaling
		var rest string
		var err error

		if params[0] == '(' {
			components, arraySuffix, newRest, tupleErr := parseTupleType(params)
			if tupleErr != nil {
				return nil, fmt.Errorf("failed to parse tuple: %v", tupleErr)
			}
			rest = newRest
			rest = skipWhitespace(rest)

			indexed := false
			if allowIndexed {
				rest, indexed = parseKeyword(rest, "indexed")
				rest = skipWhitespace(rest)
			}

			paramName := fmt.Sprintf("param%d", paramIndex)
			if len(rest) > 0 && (isAlpha(rest[0]) || isIdentifierSymbol(rest[0])) {
				paramName, rest, err = parseIdentifier(rest)
				if err != nil {
					return nil, err
				}
			}

			arg = ArgumentMarshaling{
				Name:       paramName,
				Type:       "tuple" + arraySuffix,
				Components: components,
				Indexed:    indexed,
			}
		} else {
			var typeName string
			typeName, rest, err = parseElementaryType(params)
			if err != nil {
				return nil, fmt.Errorf("failed to parse parameter type: %v", err)
			}

			rest = skipWhitespace(rest)

			indexed := false
			if allowIndexed {
				rest, indexed = parseKeyword(rest, "indexed")
				rest = skipWhitespace(rest)
			}

			paramName := fmt.Sprintf("param%d", paramIndex)
			if len(rest) > 0 && (isAlpha(rest[0]) || isIdentifierSymbol(rest[0])) {
				paramName, rest, err = parseIdentifier(rest)
				if err == nil && paramName != "" {
					rest = skipWhitespace(rest)
				}
			}

			arg = ArgumentMarshaling{
				Name:    paramName,
				Type:    typeName,
				Indexed: indexed,
			}
		}

		arguments = append(arguments, arg)
		rest = skipWhitespace(rest)

		if rest == "" {
			break
		}
		if rest[0] == ',' {
			rest = rest[1:]
			params = rest
			paramIndex++
		} else {
			break
		}
	}

	return arguments, nil
}

// ParseEvent parses an event signature into EventMarshaling
func ParseEvent(unescapedSelector string) (EventMarshaling, error) {
	unescapedSelector = skipWhitespace(unescapedSelector)

	rest, _ := parseKeyword(unescapedSelector, "event")

	name, rest, err := parseIdentifier(rest)
	if err != nil {
		return EventMarshaling{}, fmt.Errorf("failed to parse event name: %v", err)
	}

	rest = skipWhitespace(rest)

	if len(rest) == 0 || rest[0] != '(' {
		return EventMarshaling{}, fmt.Errorf("expected '(' after event name")
	}
	rest = rest[1:]

	parenCount := 1
	paramEnd := 0
	for i := 0; i < len(rest); i++ {
		if rest[i] == '(' {
			parenCount++
		} else if rest[i] == ')' {
			parenCount--
			if parenCount == 0 {
				paramEnd = i
				break
			}
		}
	}

	if parenCount != 0 {
		return EventMarshaling{}, fmt.Errorf("unbalanced parentheses in event signature")
	}

	paramsStr := rest[:paramEnd]
	arguments, err := parseParameterList(paramsStr, true)
	if err != nil {
		return EventMarshaling{}, fmt.Errorf("failed to parse event parameters: %v", err)
	}

	rest = skipWhitespace(rest[paramEnd+1:])
	_, anonymous := parseKeyword(rest, "anonymous")

	return EventMarshaling{
		Name:      name,
		Type:      "event",
		Inputs:    arguments,
		Anonymous: anonymous,
	}, nil
}

// ParseError parses an error signature into ErrorMarshaling
func ParseError(unescapedSelector string) (ErrorMarshaling, error) {
	unescapedSelector = skipWhitespace(unescapedSelector)

	rest, _ := parseKeyword(unescapedSelector, "error")

	name, rest, err := parseIdentifier(rest)
	if err != nil {
		return ErrorMarshaling{}, fmt.Errorf("failed to parse error name: %v", err)
	}

	rest = skipWhitespace(rest)

	if len(rest) == 0 || rest[0] != '(' {
		return ErrorMarshaling{}, fmt.Errorf("expected '(' after error name")
	}
	rest = rest[1:]

	parenCount := 1
	paramEnd := 0
	for i := 0; i < len(rest); i++ {
		if rest[i] == '(' {
			parenCount++
		} else if rest[i] == ')' {
			parenCount--
			if parenCount == 0 {
				paramEnd = i
				break
			}
		}
	}

	if parenCount != 0 {
		return ErrorMarshaling{}, fmt.Errorf("unbalanced parentheses in error signature")
	}

	paramsStr := rest[:paramEnd]
	arguments, err := parseParameterList(paramsStr, false)
	if err != nil {
		return ErrorMarshaling{}, fmt.Errorf("failed to parse error parameters: %v", err)
	}

	return ErrorMarshaling{
		Name:   name,
		Type:   "error",
		Inputs: arguments,
	}, nil
}

// ParseConstructor parses a constructor signature
func ParseConstructor(signature string) (ABIMarshaling, error) {
	signature = skipWhitespace(signature)

	rest, _ := parseKeyword(signature, "constructor")
	rest = skipWhitespace(rest)

	if len(rest) == 0 || rest[0] != '(' {
		return nil, fmt.Errorf("expected '(' after constructor keyword")
	}
	rest = rest[1:]

	parenCount := 1
	paramEnd := 0
	for i := 0; i < len(rest); i++ {
		if rest[i] == '(' {
			parenCount++
		} else if rest[i] == ')' {
			parenCount--
			if parenCount == 0 {
				paramEnd = i
				break
			}
		}
	}

	if parenCount != 0 {
		return nil, fmt.Errorf("unbalanced parentheses in constructor signature")
	}

	paramsStr := rest[:paramEnd]
	arguments, err := parseParameterList(paramsStr, false)
	if err != nil {
		return nil, fmt.Errorf("failed to parse constructor parameters: %v", err)
	}

	rest = skipWhitespace(rest[paramEnd+1:])

	stateMutability := "nonpayable"
	if newRest, found := parseKeyword(rest, "payable"); found {
		stateMutability = "payable"
		rest = newRest
	}

	result := make(ABIMarshaling)
	result["type"] = "constructor"
	result["inputs"] = arguments
	result["stateMutability"] = stateMutability
	return result, nil
}

// ParseFallback parses a fallback function signature
// ParseFallback parses a fallback function signature (no parameters allowed)
func ParseFallback(signature string) (ABIMarshaling, error) {
	signature = skipWhitespace(signature)

	rest, _ := parseKeyword(signature, "fallback")
	rest = skipWhitespace(rest)

	if len(rest) == 0 || rest[0] != '(' {
		return nil, fmt.Errorf("expected '(' after fallback keyword")
	}
	rest = rest[1:]

	if len(rest) == 0 || rest[0] != ')' {
		return nil, fmt.Errorf("fallback function cannot have parameters")
	}
	rest = skipWhitespace(rest[1:])

	stateMutability := "nonpayable"
	if newRest, found := parseKeyword(rest, "payable"); found {
		stateMutability = "payable"
		rest = newRest
	}

	result := make(ABIMarshaling)
	result["type"] = "fallback"
	result["stateMutability"] = stateMutability
	return result, nil
}

// ParseReceive parses a receive function signature (no parameters, always payable)
func ParseReceive(signature string) (ABIMarshaling, error) {
	signature = skipWhitespace(signature)

	rest, _ := parseKeyword(signature, "receive")
	rest = skipWhitespace(rest)

	if len(rest) == 0 || rest[0] != '(' {
		return nil, fmt.Errorf("expected '(' after receive keyword")
	}
	rest = rest[1:]

	if len(rest) == 0 || rest[0] != ')' {
		return nil, fmt.Errorf("receive function cannot have parameters")
	}
	rest = skipWhitespace(rest[1:])

	stateMutability := "payable"
	if newRest, found := parseKeyword(rest, "payable"); found {
		rest = newRest
	}

	result := make(ABIMarshaling)
	result["type"] = "receive"
	result["stateMutability"] = stateMutability
	return result, nil
}

func ParseSelector(unescapedSelector string) (SelectorMarshaling, error) {
	unescapedSelector = skipWhitespace(unescapedSelector)

	rest, _ := parseKeyword(unescapedSelector, "function")
	rest = skipWhitespace(rest)

	name, rest, err := parseIdentifier(rest)
	if err != nil {
		return SelectorMarshaling{}, fmt.Errorf("failed to parse selector '%s': %v", unescapedSelector, err)
	}

	rest = skipWhitespace(rest)

	if len(rest) == 0 || rest[0] != '(' {
		return SelectorMarshaling{}, fmt.Errorf("expected '(' after function name")
	}
	rest = rest[1:]

	parenCount := 1
	paramEnd := 0
	for i := 0; i < len(rest); i++ {
		if rest[i] == '(' {
			parenCount++
		} else if rest[i] == ')' {
			parenCount--
			if parenCount == 0 {
				paramEnd = i
				break
			}
		}
	}

	if parenCount != 0 {
		return SelectorMarshaling{}, fmt.Errorf("unbalanced parentheses in function signature")
	}

	paramsStr := rest[:paramEnd]
	fakeArgs, err := parseParameterList(paramsStr, false)
	if err != nil {
		return SelectorMarshaling{}, fmt.Errorf("failed to parse input parameters: %v", err)
	}

	rest = skipWhitespace(rest[paramEnd+1:])

	stateMutability := "nonpayable"
	if newRest, found := parseKeyword(rest, "view"); found {
		stateMutability = "view"
		rest = newRest
	} else if newRest, found := parseKeyword(rest, "pure"); found {
		stateMutability = "pure"
		rest = newRest
	} else if newRest, found := parseKeyword(rest, "payable"); found {
		stateMutability = "payable"
		rest = newRest
	}

	rest = skipWhitespace(rest)

	var outputs []ArgumentMarshaling
	if newRest, found := parseKeyword(rest, "returns"); found {
		rest = skipWhitespace(newRest)
		if len(rest) == 0 || rest[0] != '(' {
			return SelectorMarshaling{}, fmt.Errorf("expected '(' after returns keyword")
		}

		parenCount := 1
		paramEnd := 1
		for i := 1; i < len(rest); i++ {
			if rest[i] == '(' {
				parenCount++
			} else if rest[i] == ')' {
				parenCount--
				if parenCount == 0 {
					paramEnd = i
					break
				}
			}
		}

		if parenCount != 0 {
			return SelectorMarshaling{}, fmt.Errorf("unbalanced parentheses in returns clause")
		}

		returnsStr := rest[1:paramEnd]
		outputs, err = parseParameterList(returnsStr, false)
		if err != nil {
			return SelectorMarshaling{}, fmt.Errorf("failed to parse returns: %v", err)
		}

		rest = skipWhitespace(rest[paramEnd+1:])
	}

	rest = skipWhitespace(rest)
	if stateMutability == "nonpayable" {
		if newRest, found := parseKeyword(rest, "view"); found {
			stateMutability = "view"
			rest = newRest
		} else if newRest, found := parseKeyword(rest, "pure"); found {
			stateMutability = "pure"
			rest = newRest
		} else if newRest, found := parseKeyword(rest, "payable"); found {
			stateMutability = "payable"
			rest = newRest
		}
	}

	rest = skipWhitespace(rest)

	if len(rest) > 0 {
		return SelectorMarshaling{}, fmt.Errorf("failed to parse selector '%s': unexpected string '%s'", unescapedSelector, rest)
	}

	return SelectorMarshaling{
		Name:            name,
		Type:            "function",
		Inputs:          fakeArgs,
		Outputs:         outputs,
		StateMutability: stateMutability,
		Anonymous:       false,
	}, nil
}
