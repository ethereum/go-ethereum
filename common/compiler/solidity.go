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
	"io/ioutil"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	versionRegexp = regexp.MustCompile(`([0-9]+)\.([0-9]+)\.([0-9]+)`)
	spmu          = &sync.Mutex{} // protects access to mutating _solcParams variable (contents are never mutated)
	_solcParams    = []string{}   // zero-len means uninitialized, protected by spmu
)


type Contract struct {
	Code string       `json:"code"`
	Info ContractInfo `json:"info"`
}

type ContractInfo struct {
	Source          string      `json:"source"`
	Language        string      `json:"language"`
	LanguageVersion string      `json:"languageVersion"`
	CompilerVersion string      `json:"compilerVersion"`
	CompilerOptions string      `json:"compilerOptions"`
	AbiDefinition   interface{} `json:"abiDefinition"`
	UserDoc         interface{} `json:"userDoc"`
	DeveloperDoc    interface{} `json:"developerDoc"`
	Metadata        string      `json:"metadata"`
}

// Solidity contains information about the solidity compiler.
type Solidity struct {
	Path, Version, FullVersion string
}

// --combined-output format
type solcOutput struct {
	Contracts map[string]struct{ Bin, Abi, Devdoc, Userdoc, Metadata string }
	Version   string
}

func findSolcParams(solc string) []string {
	spmu.Lock()
	defer spmu.Unlock()

	if ( len(_solcParams) != 0 ) {
		return _solcParams
	} else {
		recent, _ := atLeast047(solc) // on err, we default to considering the solidity version recent, see atLeast047
		var mbmeta string
		if ( recent ) {
			mbmeta = ",metadata"
		} else {
			mbmeta = ""
		}
		_solcParams = []string{
			"--combined-json", "bin,abi,userdoc,devdoc" + mbmeta,
			"--add-std",  // include standard lib contracts
			"--optimize", // code optimizer switched on
		}
		return _solcParams
	}
}

func atLeast047(solc string) (bool, error) {
	var (
		solidity *Solidity
		major    int
		minor    int
		patch    int
		err      error
	)
	solidity, err = SolidityVersion(solc)
	if ( err != nil ) {
		return true, err
	}

	matches := versionRegexp.FindStringSubmatch( solidity.Version )
	major, err = strconv.Atoi( matches[1] )
	if ( err != nil ) {
		return true, err
	}
	minor, err = strconv.Atoi( matches[2] )
	if ( err != nil ) {
		return true, err
	}
	patch, err = strconv.Atoi( matches[3] )
	if ( err != nil ) {
		return true, err
	}
	return (major > 0 || minor > 4 || patch > 6), nil
}

// SolidityVersion runs solc and parses its version output.
func SolidityVersion(solc string) (*Solidity, error) {
	if solc == "" {
		solc = "solc"
	}
	var out bytes.Buffer
	cmd := exec.Command(solc, "--version")
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	s := &Solidity{
		Path:        cmd.Path,
		FullVersion: out.String(),
		Version:     versionRegexp.FindString(out.String()),
	}
	return s, nil
}

// CompileSolidityString builds and returns all the contracts contained within a source string.
func CompileSolidityString(solc, source string) (map[string]*Contract, error) {
	if len(source) == 0 {
		return nil, errors.New("solc: empty source string")
	}
	if solc == "" {
		solc = "solc"
	}
	solcParams := findSolcParams(solc)
	args := append(solcParams, "--")
	cmd := exec.Command(solc, append(args, "-")...)
	cmd.Stdin = strings.NewReader(source)
	return runsolc(cmd, solcParams, source)
}

// CompileSolidity compiles all given Solidity source files.
func CompileSolidity(solc string, sourcefiles ...string) (map[string]*Contract, error) {
	if len(sourcefiles) == 0 {
		return nil, errors.New("solc: no source files")
	}
	source, err := slurpFiles(sourcefiles)
	if err != nil {
		return nil, err
	}
	if solc == "" {
		solc = "solc"
	}
	solcParams := findSolcParams(solc)
	args := append(solcParams, "--")
	cmd := exec.Command(solc, append(args, sourcefiles...)...)
	return runsolc(cmd, solcParams, source)
}

func runsolc(cmd *exec.Cmd, solcParams []string, source string) (map[string]*Contract, error) {
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("solc: %v\n%s", err, stderr.Bytes())
	}
	var output solcOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, err
	}
	shortVersion := versionRegexp.FindString(output.Version)

	// Compilation succeeded, assemble and return the contracts.
	contracts := make(map[string]*Contract)
	for name, info := range output.Contracts {
		// Parse the individual compilation results.
		var abi interface{}
		if err := json.Unmarshal([]byte(info.Abi), &abi); err != nil {
			return nil, fmt.Errorf("solc: error reading abi definition (%v)", err)
		}
		var userdoc interface{}
		if err := json.Unmarshal([]byte(info.Userdoc), &userdoc); err != nil {
			return nil, fmt.Errorf("solc: error reading user doc: %v", err)
		}
		var devdoc interface{}
		if err := json.Unmarshal([]byte(info.Devdoc), &devdoc); err != nil {
			return nil, fmt.Errorf("solc: error reading dev doc: %v", err)
		}
		var metadata string
		if info.Metadata != "" {
			jstring, err := json.Marshal( string(info.Metadata) )
			if ( err != nil ) {
			      return nil, fmt.Errorf("solc: error coercing metadata to string: %v", err)
			}
		        if err := json.Unmarshal(jstring, &metadata); err != nil {
			      return nil, fmt.Errorf("solc: error reading metadata: %v", err)
			}
		}
		contracts[name] = &Contract{
			Code: "0x" + info.Bin,
			Info: ContractInfo{
				Source:          source,
				Language:        "Solidity",
				LanguageVersion: shortVersion,
				CompilerVersion: shortVersion,
				CompilerOptions: strings.Join(solcParams, " "),
				AbiDefinition:   abi,
				UserDoc:         userdoc,
				DeveloperDoc:    devdoc,
				Metadata:        metadata,
			},
		}
	}
	return contracts, nil
}

func slurpFiles(files []string) (string, error) {
	var concat bytes.Buffer
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return "", err
		}
		concat.Write(content)
	}
	return concat.String(), nil
}

// SaveInfo serializes info to the given file and returns its Keccak256 hash.
func SaveInfo(info *ContractInfo, filename string) (common.Hash, error) {
	infojson, err := json.Marshal(info)
	if err != nil {
		return common.Hash{}, err
	}
	contenthash := common.BytesToHash(crypto.Keccak256(infojson))
	return contenthash, ioutil.WriteFile(filename, infojson, 0600)
}
