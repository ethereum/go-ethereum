package abi

import (
	"fmt"
)

type SelectorMarshaling struct {
	Name   string               `json:"name"`
	Type   string               `json:"type"`
	Inputs []ArgumentMarshaling `json:"inputs"`
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
		return "", "", fmt.Errorf("empty token")
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

func parseElementaryType(unescapedSelector string) (string, string, error) {
	parsedType, rest, err := parseToken(unescapedSelector, false)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse elementary type: %v", err)
	}
	// handle arrays
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
	return parsedType, rest, nil
}

func parseCompositeType(unescapedSelector string) ([]interface{}, string, error) {
	if len(unescapedSelector) == 0 || unescapedSelector[0] != '(' {
		return nil, "", fmt.Errorf("expected '(', got %c", unescapedSelector[0])
	}
	parsedType, rest, err := parseType(unescapedSelector[1:])
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse type: %v", err)
	}
	result := []interface{}{parsedType}
	for len(rest) > 0 && rest[0] != ')' {
		parsedType, rest, err = parseType(rest[1:])
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse type: %v", err)
		}
		result = append(result, parsedType)
	}
	if len(rest) == 0 || rest[0] != ')' {
		return nil, "", fmt.Errorf("expected ')', got '%s'", rest)
	}
	return result, rest[1:], nil
}

func parseType(unescapedSelector string) (interface{}, string, error) {
	if len(unescapedSelector) == 0 {
		return nil, "", fmt.Errorf("empty type")
	}
	if unescapedSelector[0] == '(' {
		return parseCompositeType(unescapedSelector)
	} else {
		return parseElementaryType(unescapedSelector)
	}
}

func assembleArgs(args []interface{}) ([]ArgumentMarshaling, error) {
	arguments := make([]ArgumentMarshaling, 0)
	for i, arg := range args {
		// generate dummy name to avoid unmarshal issues
		name := fmt.Sprintf("name%d", i)
		if s, ok := arg.(string); ok {
			arguments = append(arguments, ArgumentMarshaling{name, s, s, nil, false})
		} else if components, ok := arg.([]interface{}); ok {
			subArgs, err := assembleArgs(components)
			if err != nil {
				return nil, fmt.Errorf("failed to assemble components: %v", err)
			}
			arguments = append(arguments, ArgumentMarshaling{name, "tuple", "tuple", subArgs, false})
		} else {
			return nil, fmt.Errorf("failed to assemble args: unexpected type %T", arg)
		}
	}
	return arguments, nil
}

// ParseSelector converts a method selector into a struct that can be JSON encoded
// and consumed by other functions in this package.
// Note, although uppercase letters are not part of the ABI spec, this function
// still accepts it as the general format is valid.
func ParseSelector(unescapedSelector string) (SelectorMarshaling, error) {
	name, rest, err := parseIdentifier(unescapedSelector)
	if err != nil {
		return SelectorMarshaling{}, fmt.Errorf("failed to parse selector '%s': %v", unescapedSelector, err)
	}
	args := []interface{}{}
	if len(rest) >= 2 && rest[0] == '(' && rest[1] == ')' {
		rest = rest[2:]
	} else {
		args, rest, err = parseCompositeType(rest)
		if err != nil {
			return SelectorMarshaling{}, fmt.Errorf("failed to parse selector '%s': %v", unescapedSelector, err)
		}
	}
	if len(rest) > 0 {
		return SelectorMarshaling{}, fmt.Errorf("failed to parse selector '%s': unexpected string '%s'", unescapedSelector, rest)
	}

	// Reassemble the fake ABI and constuct the JSON
	fakeArgs, err := assembleArgs(args)
	if err != nil {
		return SelectorMarshaling{}, fmt.Errorf("failed to parse selector: %v", err)
	}

	return SelectorMarshaling{name, "function", fakeArgs}, nil
}
