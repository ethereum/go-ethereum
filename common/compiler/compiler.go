package compiler

import (
	"fmt"
	"os/exec"
)

// An interface to denote a compiler. It implements two function.
// Compile takes in a slice of files and a series of functions made
// to represent flag options of the compiler and returns a Compile Return or an error.
// The function version() is instantiated upon the creation of the compiler to
// fill the compiler's struct with version details. It fails if there is an error in the parsing.
type Compiler interface {
	Compile(files []string, flags FlagOpts) (Return, error)
	version() error
}

// Practicing inheritance, this struct gives us access to all types of returns.
// This is written to be extendable to other compilers.
type Return struct {
	Error error
	SolcReturn
	//Enter your return struct here...e.g.
	//SerpentReturn
	//BambooReturn
	//ViperReturn
}

// Practicing inheritance, this struct allows us to easily create a simple
// interface for interacting with our potentially various compilers.
type FlagOpts struct {
	SolcFlagOpts
	//Enter your FlagOpts struct here...
}

func InitCompiler(compilerName string) (Compiler, error) {
	if _, err := exec.LookPath(compilerName); err != nil {
		return nil, fmt.Errorf("compiler: could not find %v in PATH", compilerName)
	}
	switch compilerName {
	case "solc":
		s := &Solidity{}
		if err := s.version(); err != nil {
			return nil, err
		} else {
			return s, nil
		}
	default:
		return nil, fmt.Errorf("compiler: currently does not support %v for compilation", compilerName)
	}
}
