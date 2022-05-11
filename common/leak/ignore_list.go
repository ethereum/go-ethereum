package leak

import "go.uber.org/goleak"

func IgnoreList() []goleak.Option {
	return []goleak.Option {
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/core.(*txSenderCacher).cache"),
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*recursiveTree).dispatch"),
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify._Cfunc_CFRunLoopRun"),
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/metrics.(*meterArbiter).tick"),
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/consensus/ethash.(*remoteSealer).loop"),
	}
}