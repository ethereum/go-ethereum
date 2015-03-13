package vm

type VirtualMachine interface {
	Env() Environment
	Run(context *Context, data []byte) ([]byte, error)
	Printf(string, ...interface{}) VirtualMachine
	Endl() VirtualMachine
}
