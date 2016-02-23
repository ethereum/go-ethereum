# RDB - RocksDB Shell

RDB is a NodeJS-based shell interface to RocksDB. It can also be used as a
JavaScript binding for RocksDB within a Node application.

## Setup/Compilation

### Requirements

* static RocksDB library (i.e. librocksdb.a)
* libsnappy
* node (tested onv0.10.33, no guarantees on anything else!)
* node-gyp
* python2 (for node-gyp; tested with 2.7.8)

### Installation

NOTE: If your default `python` binary is not a version of python2, add
the arguments `--python /path/to/python2` to the the `node-gyp` commands.

1. Make sure you have the static library (i.e. "librocksdb.a") in the root
directory of your rocksdb installation. If not, `cd` there and run
`make static_lib`.

2. Run `node-gyp configure` to generate the build.

3. Run `node-gyp build` to compile RDB.

## Usage

### Running the shell

Assuming everything compiled correctly, you can run the `rdb` executable
located in the root of the `tools/rdb` directory to start the shell. The file is
just a shell script that runs the node shell and loads the constructor for the
RDB object into the top-level function `RDB`.

### JavaScript API

See `API.md` for how to use RocksDB from the shell.
