# Walk-through of rpc package

## rpc package

If you go through every single files under rpc package you can see there is no import from other packages except logging or cors. This means rpc package is very modulalized and separated module from the rest in go-ethereum code base. We can incorporate easily by copying the whole directory.

In high-level there are implmentation of client and server.

### Client

- To create a client, we call [newClient](https://github.com/daywednes/go-ethereum/blob/master/rpc/client.go#L194)

## Links from other package to rpc

- from [cmd/swarm/global-store](https://github.com/daywednes/go-ethereum/blob/bca140b73dc107676c912d87f6fe9c352d5fd0d8/cmd/swarm/global-store/global_store.go#L96)
- from [node/node.go:startRPC](https://github.com/daywednes/go-ethereum/blob/bca140b73dc107676c912d87f6fe9c352d5fd0d8/node/node.go#L281) which has calls [In-Process RPC endpoint](https://github.com/daywednes/go-ethereum/blob/bca140b73dc107676c912d87f6fe9c352d5fd0d8/node/node.go#L288), call to [IPC RPC endpoint](https://github.com/daywednes/go-ethereum/blob/bca140b73dc107676c912d87f6fe9c352d5fd0d8/node/node.go#L334), [HTTP RPC endpoint](https://github.com/daywednes/go-ethereum/blob/bca140b73dc107676c912d87f6fe9c352d5fd0d8/node/node.go#L363). startRPC is called from [node.Start](https://github.com/daywednes/go-ethereum/blob/bca140b73dc107676c912d87f6fe9c352d5fd0d8/node/node.go#L162)
- [node.Start](https://github.com/daywednes/go-ethereum/blob/bca140b73dc107676c912d87f6fe9c352d5fd0d8/node/node.go#L162) is called from
  - [mobile/n.node.Start](https://github.com/daywednes/go-ethereum/blob/bca140b73dc107676c912d87f6fe9c352d5fd0d8/mobile/geth.go#L202)
  - [cmd/faucet:stack.Start()](https://github.com/daywednes/go-ethereum/blob/bca140b73dc107676c912d87f6fe9c352d5fd0d8/cmd/faucet/faucet.go#L254)
  - [cmd/utils/StartNode](https://github.com/daywednes/go-ethereum/blob/bca140b73dc107676c912d87f6fe9c352d5fd0d8/cmd/utils/cmd.go#L66) which is called from geth/main.go
