package vm

type VirtualMachine interface {
	Env() Environment
	RunClosure(*Closure) ([]byte, error)
	Depth() int
	Printf(string, ...interface{}) VirtualMachine
	Endl() VirtualMachine
}
