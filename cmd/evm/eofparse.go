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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func init() {
	jt = vm.NewPragueEOFInstructionSetForTesting()
}

var (
	jt       vm.JumpTable
	initcode = "INITCODE"
)

func eofParseAction(ctx *cli.Context) error {
	// If `--test` is set, parse and validate the reference test at the provided path.
	if ctx.IsSet(refTestFlag.Name) {
		var (
			file          = ctx.String(refTestFlag.Name)
			executedTests int
			passedTests   int
		)
		err := filepath.Walk(file, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			log.Debug("Executing test", "name", info.Name())
			passed, tot, err := executeTest(path)
			passedTests += passed
			executedTests += tot
			return err
		})
		if err != nil {
			return err
		}
		log.Info("Executed tests", "passed", passedTests, "total executed", executedTests)
		return nil
	}
	// If `--hex` is set, parse and validate the hex string argument.
	if ctx.IsSet(hexFlag.Name) {
		if _, err := parseAndValidate(ctx.String(hexFlag.Name), false); err != nil {
			return fmt.Errorf("err: %w", err)
		}
		fmt.Println("OK")
		return nil
	}
	// If neither are passed in, read input from stdin.
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(l, "#") || l == "" {
			continue
		}
		if _, err := parseAndValidate(l, false); err != nil {
			fmt.Printf("err: %v\n", err)
		} else {
			fmt.Println("OK")
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err.Error())
	}
	return nil
}

type refTests struct {
	Vectors map[string]eOFTest `json:"vectors"`
}

type eOFTest struct {
	Code          string              `json:"code"`
	Results       map[string]etResult `json:"results"`
	ContainerKind string              `json:"containerKind"`
}

type etResult struct {
	Result    bool   `json:"result"`
	Exception string `json:"exception,omitempty"`
}

func executeTest(path string) (int, int, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	var testsByName map[string]refTests
	if err := json.Unmarshal(src, &testsByName); err != nil {
		return 0, 0, err
	}
	passed, total := 0, 0
	for testsName, tests := range testsByName {
		for name, tt := range tests.Vectors {
			for fork, r := range tt.Results {
				total++
				_, err := parseAndValidate(tt.Code, tt.ContainerKind == initcode)
				if r.Result && err != nil {
					log.Error("Test failure, expected validation success", "name", testsName, "idx", name, "fork", fork, "err", err)
					continue
				}
				if !r.Result && err == nil {
					log.Error("Test failure, expected validation error", "name", testsName, "idx", name, "fork", fork, "have err", r.Exception, "err", err)
					continue
				}
				passed++
			}
		}
	}
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

func eofDumpAction(ctx *cli.Context) error {
	// If `--hex` is set, parse and validate the hex string argument.
	if ctx.IsSet(hexFlag.Name) {
		return eofDump(ctx.String(hexFlag.Name))
	}
	// Otherwise read from stdin
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(l, "#") || l == "" {
			continue
		}
		if err := eofDump(l); err != nil {
			return err
		}
		fmt.Println("")
	}
	return scanner.Err()
}

func eofDump(hexdata string) error {
	if len(hexdata) >= 2 && strings.HasPrefix(hexdata, "0x") {
		hexdata = hexdata[2:]
	}
	b, err := hex.DecodeString(hexdata)
	if err != nil {
		return fmt.Errorf("unable to decode data: %w", err)
	}
	var c vm.Container
	if err := c.UnmarshalBinary(b, false); err != nil {
		return err
	}
	fmt.Println(c.String())
	return nil
}
