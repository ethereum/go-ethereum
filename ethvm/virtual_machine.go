package ethvm

type VirtualMachine interface {
	Env() Environment
	RunClosure(*Closure) ([]byte, error)
	Depth() int
}
