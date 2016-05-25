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

package compiler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

var (
	versionRegexp = regexp.MustCompile("[0-9]+\\.[0-9]+\\.[0-9]+")
	legacyRegexp  = regexp.MustCompile("0\\.(9\\..*|1\\.[01])")
	paramsLegacy  = []string{
		"--binary",       // Request to output the contract in binary (hexadecimal).
		"file",           //
		"--json-abi",     // Request to output the contract's JSON ABI interface.
		"file",           //
		"--natspec-user", // Request to output the contract's Natspec user documentation.
		"file",           //
		"--natspec-dev",  // Request to output the contract's Natspec developer documentation.
		"file",
		"--add-std",
		"1",
	}
	paramsNew = []string{
		"--bin",      // Request to output the contract in binary (hexadecimal).
		"--abi",      // Request to output the contract's JSON ABI interface.
		"--userdoc",  // Request to output the contract's Natspec user documentation.
		"--devdoc",   // Request to output the contract's Natspec developer documentation.
		"--add-std",  // include standard lib contracts
		"--optimize", // code optimizer switched on
		"-o",         // output directory
	}
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
}

type Solidity struct {
	solcPath    string
	version     string
	fullVersion string
	legacy      bool
}

func New(solcPath string) (sol *Solidity, err error) {
	// set default solc
	if len(solcPath) == 0 {
		solcPath = "solc"
	}
	solcPath, err = exec.LookPath(solcPath)
	if err != nil {
		return
	}

	cmd := exec.Command(solcPath, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return
	}

	fullVersion := out.String()
	version := versionRegexp.FindString(fullVersion)
	legacy := legacyRegexp.MatchString(version)

	sol = &Solidity{
		solcPath:    solcPath,
		version:     version,
		fullVersion: fullVersion,
		legacy:      legacy,
	}
	glog.V(logger.Info).Infoln(sol.Info())
	return
}

func (sol *Solidity) Info() string {
	return fmt.Sprintf("%s\npath: %s", sol.fullVersion, sol.solcPath)
}

func (sol *Solidity) Version() string {
	return sol.version
}

// Compile builds and returns all the contracts contained within a source string.
func (sol *Solidity) Compile(source string) (map[string]*Contract, error) {
	// Short circuit if no source code was specified
	if len(source) == 0 {
		return nil, errors.New("solc: empty source string")
	}
	// Create a safe place to dump compilation output
	wd, err := ioutil.TempDir("", "solc")
	if err != nil {
		return nil, fmt.Errorf("solc: failed to create temporary build folder: %v", err)
	}
	defer os.RemoveAll(wd)

	// Assemble the compiler command, change to the temp folder and capture any errors
	stderr := new(bytes.Buffer)

	var params []string
	if sol.legacy {
		params = paramsLegacy
	} else {
		params = paramsNew
		params = append(params, wd)
	}
	compilerOptions := strings.Join(params, " ")

	cmd := exec.Command(sol.solcPath, params...)
	cmd.Stdin = strings.NewReader(source)
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("solc: %v\n%s", err, string(stderr.Bytes()))
	}
	// Sanity check that something was actually built
	matches, _ := filepath.Glob(filepath.Join(wd, "*.bin*"))
	if len(matches) < 1 {
		return nil, fmt.Errorf("solc: no build results found")
	}
	// Compilation succeeded, assemble and return the contracts
	contracts := make(map[string]*Contract)
	for _, path := range matches {
		_, file := filepath.Split(path)
		base := strings.Split(file, ".")[0]

		// Parse the individual compilation results (code binary, ABI definitions, user and dev docs)
		var binary []byte
		binext := ".bin"
		if sol.legacy {
			binext = ".binary"
		}
		if binary, err = ioutil.ReadFile(filepath.Join(wd, base+binext)); err != nil {
			return nil, fmt.Errorf("solc: error reading compiler output for code: %v", err)
		}

		var abi interface{}
		if blob, err := ioutil.ReadFile(filepath.Join(wd, base+".abi")); err != nil {
			return nil, fmt.Errorf("solc: error reading abi definition: %v", err)
		} else if err = json.Unmarshal(blob, &abi); err != nil {
			return nil, fmt.Errorf("solc: error parsing abi definition: %v", err)
		}

		var userdoc interface{}
		if blob, err := ioutil.ReadFile(filepath.Join(wd, base+".docuser")); err != nil {
			return nil, fmt.Errorf("solc: error reading user doc: %v", err)
		} else if err = json.Unmarshal(blob, &userdoc); err != nil {
			return nil, fmt.Errorf("solc: error parsing user doc: %v", err)
		}

		var devdoc interface{}
		if blob, err := ioutil.ReadFile(filepath.Join(wd, base+".docdev")); err != nil {
			return nil, fmt.Errorf("solc: error reading dev doc: %v", err)
		} else if err = json.Unmarshal(blob, &devdoc); err != nil {
			return nil, fmt.Errorf("solc: error parsing dev doc: %v", err)
		}
		// Assemble the final contract
		contracts[base] = &Contract{
			Code: "0x" + string(binary),
			Info: ContractInfo{
				Source:          source,
				Language:        "Solidity",
				LanguageVersion: sol.version,
				CompilerVersion: sol.version,
				CompilerOptions: compilerOptions,
				AbiDefinition:   abi,
				UserDoc:         userdoc,
				DeveloperDoc:    devdoc,
			},
		}
	}
	return contracts, nil
}

func SaveInfo(info *ContractInfo, filename string) (contenthash common.Hash, err error) {
	infojson, err := json.Marshal(info)
	if err != nil {
		return
	}
	contenthash = common.BytesToHash(crypto.Keccak256(infojson))
	err = ioutil.WriteFile(filename, infojson, 0600)
	return
}
