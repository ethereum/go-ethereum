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
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/urfave/cli/v2"
)

func init() {
	jt = vm.NewPragueEOFInstructionSetForTesting()
}

var (
	jt       vm.JumpTable
	errorMap = map[string]int{
		io.ErrUnexpectedEOF.Error():     1,
		vm.ErrInvalidMagic.Error():      2,
		vm.ErrInvalidVersion.Error():    3,
		vm.ErrMissingTypeHeader.Error(): 4,
		vm.ErrInvalidTypeSize.Error():   5,
		vm.ErrMissingCodeHeader.Error(): 6,
		//vm.ErrInvalidCodeHeader.Error():      7,
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
		//vm.ErrConflictingStack.Error():       19,
		//vm.ErrInvalidBranchCount.Error():     20,
		vm.ErrInvalidOutputs.Error():         21,
		vm.ErrInvalidMaxStackHeight.Error():  22,
		vm.ErrInvalidCodeTermination.Error(): 23,
		vm.ErrUnreachableCode.Error():        24,
	}
	initcode = "INITCODE"
)

type RefTests struct {
	Vectors map[string]EOFTest `json:"vectors"`
}

type EOFTest struct {
	Code          string              `json:"code"`
	Results       map[string]etResult `json:"results"`
	ContainerKind string              `json:"containerKind"`
}

type etResult struct {
	Result    bool   `json:"result"`
	Exception string `json:"exception,omitempty"`
}

func eofParser(ctx *cli.Context) error {
	// If `--hex` is set, parse and validate the hex string argument.
	if ctx.IsSet(HexFlag.Name) {
		if _, err := parseAndValidate(ctx.String(HexFlag.Name), false); err != nil {
			if err2 := errors.Unwrap(err); err2 != nil {
				err = err2
			}
			return fmt.Errorf("err(%d): %w", errorMap[err.Error()], err)
		}
		fmt.Println("OK")
		return nil
	}

	// If `--test` is set, parse and validate the reference test at the provided path.
	if ctx.IsSet(RefTestFlag.Name) {
		var (
			file          = ctx.String(RefTestFlag.Name)
			executedTests atomic.Int32
			passedTests   atomic.Int32
		)
		if info, err := os.Stat(file); err != nil {
			return err
		} else if !info.IsDir() {
			src, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			_, _, err = ExecuteTest(src)
			return err
		} else {
			err = filepath.Walk(file, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				fmt.Printf("Executing Tests: %v\n", info.Name())
				src, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				passed, total, err := ExecuteTest(src)
				passedTests.Add(int32(passed))
				executedTests.Add(int32(total))
				return err
			})
			if err != nil {
				return err
			}
			fmt.Printf("Passed %v tests out of %v\n", passedTests.Load(), executedTests.Load())
			return nil
		}
	}

	// If neither are passed in, read input from stdin.
	reader := bufio.NewReaderSize(os.Stdin, 1024*1024)
	t, err := reader.ReadString('\n')
	for err == nil {
		l := len(t)
		if l == 0 || t[0] == '#' {
			continue
		}
		if t[l-1] == '\n' {
			t = t[:l-1] // remove newline
		}
		if _, err := parseAndValidate(t, false); err != nil {
			if err2 := errors.Unwrap(err); err2 != nil {
				err = err2
			}
			fmt.Printf("err(%d): %v\n", errorMap[err.Error()], err)
		} else {
			fmt.Println("OK")
		}
		t, err = reader.ReadString('\n')
	}
	println(err.Error())

	return nil
}

func ExecuteTest(src []byte) (int, int, error) {
	var testsByName map[string]RefTests
	if err := json.Unmarshal(src, &testsByName); err != nil {
		return 0, 0, err
	}
	passed, total := 0, 0
	for testsName, tests := range testsByName {
		for name, tt := range tests.Vectors {
			for fork, r := range tt.Results {
				total++
				// TODO(matt): all tests currently run against
				// shanghai EOF, add support for custom forks.
				_, err := parseAndValidate(tt.Code, tt.ContainerKind == initcode)
				if err2 := errors.Unwrap(err); err2 != nil {
					err = err2
				}
				if r.Result && err != nil {
					fmt.Fprintf(os.Stderr, "%s %s, %s: expected success, got %v\n", testsName, name, fork, err)
					continue
				}
				if !r.Result && err == nil {
					fmt.Fprintf(os.Stderr, "%s %s, %s: expected error %s, got %v\n", testsName, name, fork, r.Exception, err)
					continue
				}
				/*
					// TODO (MariusVanDerWijden) reenable once tests have a decent error format
					if !r.Result && err != nil && r.Exception != err.Error() {
						fmt.Fprintf(os.Stderr, "%s, %s: expected error %d, got: err(%d): %v\n", name, fork, r.Exception, errorMap[err.Error()], err)
						continue
					}
				*/
				passed++
			}
		}
	}
	fmt.Printf("%d/%d tests passed.\n", passed, total)
	return passed, total, nil
}

func parseAndValidate(s string, isInitCode bool) (*vm.Container, error) {
	if len(s) >= 2 && strings.HasPrefix(s, "0x") {
		s = s[2:]
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("unable to decode data: %w", err)
	}
	return parse(b, isInitCode)
}

func parse(b []byte, isInitCode bool) (*vm.Container, error) {
	var c vm.Container
	if err := c.UnmarshalBinary(b, isInitCode); err != nil {
		return nil, err
	}
	if err := c.ValidateCode(&jt, isInitCode); err != nil {
		return nil, err
	}
	return &c, nil
}

func eofDump(ctx *cli.Context) error {
	// If `--hex` is set, parse and validate the hex string argument.
	if ctx.IsSet(HexFlag.Name) {
		s := ctx.String(HexFlag.Name)
		if len(s) >= 2 && strings.HasPrefix(s, "0x") {
			s = s[2:]
		}
		b, err := hex.DecodeString(s)
		if err != nil {
			return fmt.Errorf("unable to decode data: %w", err)
		}
		var c vm.Container
		if err := c.UnmarshalBinary(b, false); err != nil {
			return err
		}
		fmt.Print(c.String())
		return nil
	}
	return nil
}
