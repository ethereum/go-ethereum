## Installing using pacman

The `geth` package is available from the [community repo](https://www.archlinux.org/packages/community/x86_64/geth/).

You can install it using

```shell
pacman -S geth
```

## Installing from source
Install dependencies
```shell
pacman -S git go gcc
```

Download and build geth
```shell
git clone https://github.com/ethereum/go-ethereum
cd go-ethereum
make geth
```