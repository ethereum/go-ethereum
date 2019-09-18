<!-- TODO: Read more -->

## Next steps

<!-- I like the idea of having a Getting started section. You may want to describe a few ways to get your feet wet with Geth (fast sync with a test net, dev mode, start a private network). -->

### Fast sync a test network

<!-- TODO: What happens if you don't specify? -->

You learned about the `--syncmode` argument above, but you can also specify the network to sync with, by adding it as an argument. For example to fast sync with the Rinkeby test network:

```shell
geth --syncmode "fast" --rinkeby
```

To check the status of the sync, open another terminal, attach to the geth process, and open a console with the command below:

<!-- TODO: Figuring out what to attach to seems difficult -->

```shell
geth attach ipc:{FILL_THIS}
```

Then at the `>` prompt, run the command below to the current state of the sync operation:

```shell
eth.syncing
```

## Dev mode

Geth has a development mode which sets up a single node Ethereum test network along with a number of options optimized for local development. Enable it with the `--dev` argument.

```shell
geth --dev
```

<!-- TODO: Figuring out what to attach to seems difficult -->

```shell
geth attach ipc:{FILL_THIS}
```

<!-- TODO: Then? -->

## Start a private network
