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

package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/urfave/cli/v2"
)

func init() {
	jt = vm.NewShanghaiEOFInstructionSetForTesting()
}

var (
	jt       vm.JumpTable
	errorMap = map[string]int{
		io.ErrUnexpectedEOF.Error():          1,
		vm.ErrInvalidMagic.Error():           2,
		vm.ErrInvalidVersion.Error():         3,
		vm.ErrMissingTypeHeader.Error():      4,
		vm.ErrInvalidTypeSize.Error():        5,
		vm.ErrMissingCodeHeader.Error():      6,
		vm.ErrInvalidCodeHeader.Error():      7,
		vm.ErrMissingDataHeader.Error():      8,
		vm.ErrMissingTerminator.Error():      9,
		vm.ErrTooManyInputs.Error():          10,
		vm.ErrTooManyOutputs.Error():         11,
		vm.ErrTooLargeMaxStackHeight.Error(): 12,
		vm.ErrInvalidCodeSize.Error():        13,
		vm.ErrInvalidContainerSize.Error():   14,
		vm.ErrUndefinedInstruction.Error():   15,
		vm.ErrTruncatedImmediate.Error():     16,
		vm.ErrInvalidSectionArgument.Error(): 17,
		vm.ErrInvalidJumpDest.Error():        18,
		vm.ErrConflictingStack.Error():       19,
		vm.ErrInvalidBranchCount.Error():     20,
		vm.ErrInvalidOutputs.Error():         21,
		vm.ErrInvalidMaxStackHeight.Error():  22,
		vm.ErrInvalidCodeTermination.Error(): 23,
		vm.ErrUnreachableCode.Error():        24,
	}
)

type EOFTest struct {
	Code    string              `json:"code"`
	Results map[string]etResult `json:"results"`
}

type etResult struct {
	Result    bool `json:"result"`
	Exception int  `json:"exception,omitempty"`
}

func eofParser(ctx *cli.Context) error {
	// If `--hex` is set, parse and validate the hex string argument.
	if ctx.IsSet(HexFlag.Name) {
		if _, err := parseAndValidate(ctx.String(HexFlag.Name)); err != nil {
			if err2 := errors.Unwrap(err); err2 != nil {
				err = err2
			}
			return fmt.Errorf("err(%d): %w", errorMap[err.Error()], err)
		}
		fmt.Println("ok.")
		return nil
	}

	// If `--test` is set, parse and validate the reference test at the provided path.
	if ctx.IsSet(RefTestFlag.Name) {
		src, err := os.ReadFile(ctx.String(RefTestFlag.Name))
		if err != nil {
			return err
		}
		var tests map[string]EOFTest
		if err = json.Unmarshal(src, &tests); err != nil {
			return err
		}
		passed, total := 0, 0
		for name, tt := range tests {
			for fork, r := range tt.Results {
				total++
				// TODO(matt): all tests currently run against
				// shanghai EOF, add support for custom forks.
				_, err := parseAndValidate(tt.Code)
				if err2 := errors.Unwrap(err); err2 != nil {
					err = err2
				}
				if r.Result && err != nil {
					fmt.Fprintf(os.Stderr, "%s, %s: expected success, got %v\n", name, fork, err)
					continue
				}
				if !r.Result && err == nil {
					fmt.Fprintf(os.Stderr, "%s, %s: expected error %d, got %v\n", name, fork, r.Exception, err)
					continue
				}
				if !r.Result && err != nil && r.Exception != errorMap[err.Error()] {
					fmt.Fprintf(os.Stderr, "%s, %s: expected error %d, got: err(%d): %v\n", name, fork, r.Exception, errorMap[err.Error()], err)
					continue
				}
				passed++
			}
		}
		fmt.Printf("%d/%d tests passed.\n", passed, total)
		return nil
	}

	// If neither are passed in, read input from stdin.
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		if len(t) == 0 || t[0] == '#' {
			continue
		}
		if _, err := parseAndValidate(t); err != nil {
			if err2 := errors.Unwrap(err); err2 != nil {
				err = err2
			}
			fmt.Fprintf(os.Stderr, "err(%d): %v\n", errorMap[err.Error()], err)
		}
	}

	return nil
}

func parseAndValidate(s string) (*vm.Container, error) {
	if len(s) >= 2 && strings.HasPrefix(s, "0x") {
		s = s[2:]
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("unable to decode data: %w", err)
	}
	var c vm.Container
	if err := c.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	if err := c.ValidateCode(&jt); err != nil {
		return nil, err
	}
	return &c, nil
}
