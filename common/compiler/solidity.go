package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

const (
	flair           = "Christian <c@ethdev.com> and Lefteris <lefteris@ethdev.com> (c) 2014-2015"
	languageVersion = "0"
)

var (
	versionRegExp = regexp.MustCompile("[0-9]+.[0-9]+.[0-9]+")
	params        = []string{
		"--binary",       // Request to output the contract in binary (hexadecimal).
		"file",           //
		"--json-abi",     // Request to output the contract's JSON ABI interface.
		"file",           //
		"--natspec-user", // Request to output the contract's Natspec user documentation.
		"file",           //
		"--natspec-dev",  // Request to output the contract's Natspec developer documentation.
		"file",
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
	AbiDefinition   interface{} `json:"abiDefinition"`
	UserDoc         interface{} `json:"userDoc"`
	DeveloperDoc    interface{} `json:"developerDoc"`
}

type Solidity struct {
	solcPath string
	version  string
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

	version := versionRegExp.FindString(out.String())
	sol = &Solidity{
		solcPath: solcPath,
		version:  version,
	}
	glog.V(logger.Info).Infoln(sol.Info())
	return
}

func (sol *Solidity) Info() string {
	return fmt.Sprintf("solc v%s\nSolidity Compiler: %s\n%s", sol.version, sol.solcPath, flair)
}

func (sol *Solidity) Compile(source string) (contract *Contract, err error) {

	if len(source) == 0 {
		err = fmt.Errorf("empty source")
		return
	}

	wd, err := ioutil.TempDir("", "solc")
	if err != nil {
		return
	}
	defer os.RemoveAll(wd)

	in := strings.NewReader(source)
	var out bytes.Buffer
	// cwd set to temp dir
	cmd := exec.Command(sol.solcPath, params...)
	cmd.Dir = wd
	cmd.Stdin = in
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("solc error: %v", err)
		return
	}

	matches, _ := filepath.Glob(wd + "/*.binary")
	if len(matches) < 1 {
		err = fmt.Errorf("solc error: missing code output")
		return
	}
	if len(matches) > 1 {
		err = fmt.Errorf("multi-contract sources are not supported")
		return
	}
	_, file := filepath.Split(matches[0])
	base := strings.Split(file, ".")[0]

	codeFile := path.Join(wd, base+".binary")
	abiDefinitionFile := path.Join(wd, base+".abi")
	userDocFile := path.Join(wd, base+".docuser")
	developerDocFile := path.Join(wd, base+".docdev")

	code, err := ioutil.ReadFile(codeFile)
	if err != nil {
		err = fmt.Errorf("error reading compiler output for code: %v", err)
		return
	}
	abiDefinitionJson, err := ioutil.ReadFile(abiDefinitionFile)
	if err != nil {
		err = fmt.Errorf("error reading compiler output for abiDefinition: %v", err)
		return
	}
	var abiDefinition interface{}
	err = json.Unmarshal(abiDefinitionJson, &abiDefinition)

	userDocJson, err := ioutil.ReadFile(userDocFile)
	if err != nil {
		err = fmt.Errorf("error reading compiler output for userDoc: %v", err)
		return
	}
	var userDoc interface{}
	err = json.Unmarshal(userDocJson, &userDoc)

	developerDocJson, err := ioutil.ReadFile(developerDocFile)
	if err != nil {
		err = fmt.Errorf("error reading compiler output for developerDoc: %v", err)
		return
	}
	var developerDoc interface{}
	err = json.Unmarshal(developerDocJson, &developerDoc)

	contract = &Contract{
		Code: string(code),
		Info: ContractInfo{
			Source:          source,
			Language:        "Solidity",
			LanguageVersion: languageVersion,
			CompilerVersion: sol.version,
			AbiDefinition:   abiDefinition,
			UserDoc:         userDoc,
			DeveloperDoc:    developerDoc,
		},
	}

	return
}

func ExtractInfo(contract *Contract, filename string) (contenthash common.Hash, err error) {
	contractInfo, err := json.Marshal(contract.Info)
	if err != nil {
		return
	}
	contenthash = common.BytesToHash(crypto.Sha3(contractInfo))
	err = ioutil.WriteFile(filename, contractInfo, 0600)
	return
}
