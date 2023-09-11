package leak

import "go.uber.org/goleak"

func IgnoreList() []goleak.Option {
	return []goleak.Option{
		// a list of goroutne leaks that hard to fix due to external dependencies or too big refactoring needed
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/core.(*txSenderCacher).cache"),
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*recursiveTree).dispatch"),
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*recursiveTree).internal"),
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*nonrecursiveTree).dispatch"),
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*nonrecursiveTree).internal"),
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify._Cfunc_CFRunLoopRun"),

		// todo: this leaks should be fixed
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/metrics.(*meterArbiter).tick"),
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/consensus/ethash.(*remoteSealer).loop"),
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/core.(*BlockChain).updateFutureBlocks"),
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/core/state/snapshot.(*diskLayer).generate"),
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/accounts/abi/bind/backends.nullSubscription.func1"),
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/eth/filters.(*EventSystem).eventLoop"),
	}
}
