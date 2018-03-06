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

// Package compiler wraps the Solidity compiler executable (solc).
package compiler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

var versionRegexp = regexp.MustCompile(`([0-9]+)\.([0-9]+)\.([0-9]+)`)

// Initialize a versioned Solc compiler.
func InitSolc(command string) (*Solidity, error) {
	if command == "" {
		command = "solc"
	}
	if _, err := exec.LookPath(command); err != nil {
		return nil, fmt.Errorf("compiler: could not find %v in PATH", command)
	}
	s := &Solidity{NamedCmd: command}
	if err := s.version(); err != nil {
		return nil, err
	}
	return s, nil

}

//The following represents solidity outputs from the compiler that we're interested in
type SolcReturn struct {
	Warning   string
	Version   string               `json:"version"`
	Contracts map[string]SolcItems `json:"contracts"`
}

//The key return items to enable unmarshalling from the returns from the compiler
type SolcItems struct {
	Bin      string `json:"bin"`
	Abi      string `json:"abi"`
	DevDoc   string `json:"devdoc"`
	UserDoc  string `json:"userdoc"`
	Metadata string `json:"metadata"`
}

// Solidity contains information about the solidity compiler.
type Solidity struct {
	NamedCmd, Path, Version, FullVersion string
	Major, Minor, Patch                  int
}

//This is a template to define our inputs for the compiler flags
type SolcFlagOpts struct {
	// (Required) what to get in the output, can be any combination of [abi, bin, userdoc, devdoc, metadata]
	// abi: application binary interface. Necessary for interaction with contracts.
	// bin: binary bytecode. Necessary for creating and deploying and interacting with contracts.
	// userdoc: natspec for users.
	// devdoc: natspec for devs.
	// metadata: contract metadata.
	CombinedOutput []string
	// (Optional) Direct string of library address mappings.
	//  Syntax: <libraryName>:<address>,<libraryName>:<address>
	//  Address is interpreted as a hex string optionally prefixed by 0x.
	Libraries map[string]common.Address
	// (Optional) Remappings, see https://solidity.readthedocs.io/en/latest/layout-of-source-files.html#use-in-actual-compilers
	// Syntax: <remoteName>=<localName>
	Remappings []string
	// (Optional) if true, enable standard library contracts
	StdLib bool
	// (Optional) if true, optimizes solidity code
	Optimize bool
	// (Optional) the number of optimization runs to run on solidity
	OptimizeRuns uint64
	// (Optional) For anything else we may have missed, if filled will default override other flags.
	Exec string
}

// SolidityVersion runs solc and parses its version output.
func (s *Solidity) version() error {
	var out bytes.Buffer
	cmd := exec.Command(s.NamedCmd, "--version")
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return err
	}
	matches := versionRegexp.FindStringSubmatch(out.String())
	if len(matches) != 4 {
		return fmt.Errorf("%v: can't parse version %q", s.NamedCmd, out.String())
	}
	s = &Solidity{Path: cmd.Path, FullVersion: out.String(), Version: matches[0]}
	if s.Major, err = strconv.Atoi(matches[1]); err != nil {
		return err
	}
	if s.Minor, err = strconv.Atoi(matches[2]); err != nil {
		return err
	}
	if s.Patch, err = strconv.Atoi(matches[3]); err != nil {
		return err
	}
	return nil
}

// Compiles a series of files using the solidity compiler
func (s *Solidity) Compile(flags SolcFlagOpts, files ...string) (Return, error) {

	if reflect.DeepEqual(flags.SolcFlagOpts, (SolcFlagOpts{})) {
		flags.defaultSolcFlagOpts(s)
	}

	return s.execute(flags.assembleSolcCommand(files...)...)
}

func (s *Solidity) Link(libs map[string]common.Address, binary string) (string, error) {
	var refinedBinary bytes.Buffer
	var stderr bytes.Buffer

	buf := bytes.NewBufferString(binary)
	linkCmd := exec.Command("solc", "--link", "--libraries", stringifyLibs(libs))

	linkCmd.Stdin = buf
	linkCmd.Stderr = &stderr
	linkCmd.Stdout = &refinedBinary

	linkCmd.Start()
	linkCmd.Wait()

	if stderr.String() != "" {
		return "", errors.New(stderr.String())
	}

	return refinedBinary.String(), nil
}

func stringifyLibs(libs map[string]common.Address) string {
	var combinedLibs []string
	for x, y := range libs {
		combinedLibs = append(combinedLibs, x+":"+y.String())
	}
	return strings.Join(combinedLibs, ",")
}

func (f SolcFlagOpts) assembleSolcCommand(files ...string) (command []string) {

	switch {
	case f.Exec != "":
		command = append(command, f.Exec)
	default:
		if len(f.Remappings) > 0 {
			command = append(command, strings.Join(f.Remappings, " "))
		}
		if len(f.CombinedOutput) > 0 {
			command = append(command, "--combined-json", strings.Join(f.CombinedOutput, ","))
		}
		if len(f.Libraries) > 0 {
			command = append(command, []string{"--libraries", stringifyLibs(f.Libraries)}...)
		}
		if f.Optimize {
			command = append(command, "--optimize")
		}
		if f.StdLib {
			command = append(command, "--std-lib")
		}
		if f.OptimizeRuns != 0 {
			command = append(command, "--optimize-runs", strconv.FormatUint(f.OptimizeRuns, 10))
		}
	}
	return append(command, files...)
}

func (f *SolcFlagOpts) defaultSolcFlagOpts(s *Solidity) {
	f = &SolcFlagOpts{
		CombinedOutput: []string{"bin", "abi", "userdoc", "devdoc"},
		StdLib:         true,
		Optimize:       true,
	}

	if s.Major >= 0 && s.Minor >= 4 && s.Patch > 6 {
		f.CombinedOutput = append(f.CombinedOutput, "metadata")
	}

	return
}

func (s *Solidity) execute(flagsAndFiles ...string) (SolcReturn, error) {
	var stderr, stdout bytes.Buffer
	var output SolcReturn

	cmd := exec.Command(s.NamedCmd, flagsAndFiles...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return Return{}, fmt.Errorf("%v: %v\n%s", s.NamedCmd, err, stderr.Bytes())
	}

	buf := stdout.Bytes()

	if err := json.Unmarshal(buf, &output); err != nil {
		return Return{}, err
	}

	output.Warning = string(stderr.Bytes())

	return output, nil
}
