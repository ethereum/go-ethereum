# Go Ethereum 1.8 with Timing

## Compile
Steps are the same as compiling the official Go Ethereum.

1. Install `git`, `gcc`and `go`.
2. Run `make geth` to compile.

## Run

Let's say the log file is `/path/to/file.txt`. Add `timing.output` flag in the front of the argument.

Example:

```shell
geth --timing.output=/path/to/file.txt
```

```shell
geth --timing.output=/path/to/file.txt --otherarguments...
```

## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.
