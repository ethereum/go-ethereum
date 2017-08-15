package compiler

type Compiler interface {
	PrepareCommand(files ...string) error
	Compile(flags ...func() string) (string, error)
}
