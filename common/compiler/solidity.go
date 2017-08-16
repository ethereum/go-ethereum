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
	"os"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

var versionRegexp = regexp.MustCompile(`([0-9]+)\.([0-9]+)\.([0-9]+)`)

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

// Custom UnmarshalJSON is needed for the sake of capturing the warnings that can pop up
// in the compiler while still maintaining the results of the compilation.
func (ret *SolcReturn) UnmarshalJSON(data []byte) (err error) {
	trimmedOutput := bytes.TrimSpace(data)
	jsonBeginsCertainly := bytes.Index(trimmedOutput, []byte(`{"contracts":`))

	if jsonBeginsCertainly > 0 {
		ret.Warning = string(trimmedOutput[:jsonBeginsCertainly])
		trimmedOutput = trimmedOutput[jsonBeginsCertainly:]
	}

	err = json.Unmarshal(trimmedOutput, &ret)

	return
}

func (ret SolcReturn) blend(other SolcReturn) (SolcReturn, error) {
	for str, items := range other.Contracts {
		if _, taken := ret.Contracts[str]; taken {
			return SolcReturn{}, fmt.Errorf("solc: there was an issue in blending bin and sol files, please try them separately")
		} else {
			ret.Contracts[str] = items
		}
	}
	return ret, nil
}

// Solidity contains information about the solidity compiler.
type Solidity struct {
	Path, Version, FullVersion string
	Major, Minor, Patch        int
}

//This is a template to define our inputs for the compiler flags
type SolcFlagOpts struct {
	// (Optional) what to get in the output, can be any combination of [abi, bin, userdoc, devdoc, metadata]
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

func (s *Solidity) defaultFlagOpts() (f SolcFlagOpts) {
	f = SolcFlagOpts{
		CombinedOutput: []string{"bin", "abi", "userdoc", "devdoc"},
		StdLib:         true,
		Optimize:       true,
	}

	if s.Major >= 0 && s.Minor >= 4 && s.Patch > 6 {
		f.CombinedOutput = append(f.CombinedOutput, "metadata")
	}

	return
}

// SolidityVersion runs solc and parses its version output.
func (s *Solidity) version() error {
	var out bytes.Buffer
	cmd := exec.Command("solc", "--version")
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return err
	}
	matches := versionRegexp.FindStringSubmatch(out.String())
	if len(matches) != 4 {
		return fmt.Errorf("can't parse solc version %q", out.String())
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
func (s *Solidity) Compile(files []string, flags FlagOpts) (Return, error) {

	if reflect.DeepEqual(flags.SolcFlagOpts, (SolcFlagOpts{})) {
		flags.SolcFlagOpts = s.defaultFlagOpts()
	}

	//check files for .bin extension for linking addresses
	//separate .sol and .bin files
	//link .bins separately
	solFiles, binFiles, err := s.sortAndValidateFiles(files)
	if err != nil {
		return Return{}, err
	}

	// assemble commands and execute
	var binResults SolcReturn
	if len(binFiles) > 0 {
		solcExecute := flags.assembleSolcCommand(true, binFiles...)
		binResults, err = s.executeSolc(solcExecute...)
		if err != nil {
			return Return{}, err
		}
	}

	var solResults SolcReturn
	if len(solFiles) > 0 {
		solcExecute := flags.assembleSolcCommand(false, solFiles...)
		solResults, err = s.executeSolc(solcExecute...)
		if err != nil {
			return Return{}, err
		}
	}

	// blend the two results, even if one of them is empty (more efficient this way)
	ret, err := solResults.blend(binResults)
	return Return{SolcReturn: ret}, err
}

func (f SolcFlagOpts) assembleSolcCommand(binary bool, files ...string) (command []string) {
	stringifyLibs := func(libs map[string]common.Address) []string {
		var combinedLibs []string
		for x, y := range f.Libraries {
			combinedLibs = append(combinedLibs, x+":"+y.String())
		}
		return combinedLibs
	}

	switch {
	case f.Exec != "":
		command = append(command, f.Exec)
	case binary:
		if len(f.Libraries) > 0 {
			combinedLibs := stringifyLibs(f.Libraries)
			command = append(command, "--link --libraries")
			command = append(command, strings.Join(combinedLibs, ","))
		}
	default:
		if len(f.Remappings) > 0 {
			command = append(command, strings.Join(f.Remappings, " "))
		}
		if len(f.CombinedOutput) > 0 {
			command = append(command, "--combined-json", strings.Join(f.CombinedOutput, ","))
		}
		if len(f.Libraries) > 0 {
			combinedLibs := stringifyLibs(f.Libraries)
			command = append(command, "--link --libraries")
			command = append(command, strings.Join(combinedLibs, ","))
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

// A utility function to sort .sol and .bin files into separate slices
func (s *Solidity) sortAndValidateFiles(files []string) ([]string, []string, error) {
	var solFiles []string
	var binFiles []string
	if len(files) == 0 {
		return nil, nil, errors.New("solc: no source files")
	}
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("solc: could not find file %v", file)
		}
		switch path.Ext(file) {
		case ".sol":
			solFiles = append(solFiles, file)
		case ".bin":
			binFiles = append(binFiles, file)
		default:
			return nil, nil, fmt.Errorf("solc: unexpected file extension found during compilation: %v", file)
		}
	}
	return solFiles, binFiles, nil
}

func (s *Solidity) executeSolc(flagsAndFiles ...string) (output SolcReturn, err error) {
	var stderr, stdout bytes.Buffer

	cmd := exec.Command("solc", flagsAndFiles...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err = cmd.Run(); err != nil {
		return SolcReturn{}, fmt.Errorf("solc: %v\n%s", err, stderr.Bytes())
	}

	if err = json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return SolcReturn{}, err
	}

	return
}
