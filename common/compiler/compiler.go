package compiler

// An interface to denote a version specific compiler.
// The function version() is instantiated upon the creation of the compiler to
// fill the compiler's struct with version details. It fails if there is an error in the parsing.
type Compiler interface {
	Compile(flags FlagOpts, files ...string) (Return, error)
	version() error
}

// This struct via embedding gives us access to all types of returns.
// This is written to be extendable to other compilers.
type Return struct {
	Typ CompilerType
	SolcReturn
	//Enter your return struct here...e.g.
	//SerpentReturn
	//BambooReturn
	//ViperReturn
}

// This struct allows us to easily create a simple interface for
// interacting with our potentially various compilers.
type FlagOpts struct {
	SolcFlagOpts
	//Enter your FlagOpts struct here...
}

type CompilerType byte

const (
	Solc CompilerType = iota
	// Serpent
	// Bamboo
	// Viper
)
