# Go sigar [![Build Status](https://travis-ci.org/elastic/gosigar.svg?branch=master)](https://travis-ci.org/elastic/gosigar) [![Build status](https://ci.appveyor.com/api/projects/status/4yh6sa7u97ek5uib/branch/master?svg=true)](https://ci.appveyor.com/project/elastic-beats/gosigar/branch/master)


## Overview

Go sigar is a golang implementation of the
[sigar API](https://github.com/hyperic/sigar).  The Go version of
sigar has a very similar interface, but is being written from scratch
in pure go/cgo, rather than cgo bindings for libsigar.

## Test drive

    $ go get github.com/elastic/gosigar
    $ cd $GOPATH/src/github.com/elastic/gosigar/examples/ps
    $ go build
    $ ./ps

## Supported platforms

The features vary by operating system.

| Feature         | Linux | Darwin | Windows | OpenBSD | FreeBSD |
|-----------------|:-----:|:------:|:-------:|:-------:|:-------:|
| Cpu             |   X   |    X   |    X    |    X    |    X    |
| CpuList         |   X   |    X   |         |    X    |    X    |
| FDUsage         |   X   |        |         |         |    X    |
| FileSystemList  |   X   |    X   |    X    |    X    |    X    |
| FileSystemUsage |   X   |    X   |    X    |    X    |    X    |
| HugeTLBPages    |   X   |        |         |         |         |
| LoadAverage     |   X   |    X   |         |    X    |    X    |
| Mem             |   X   |    X   |    X    |    X    |    X    |
| ProcArgs        |   X   |    X   |    X    |         |    X    |
| ProcEnv         |   X   |    X   |         |         |    X    |
| ProcExe         |   X   |    X   |         |         |    X    |
| ProcFDUsage     |   X   |        |         |         |    X    |
| ProcList        |   X   |    X   |    X    |         |    X    |
| ProcMem         |   X   |    X   |    X    |         |    X    |
| ProcState       |   X   |    X   |    X    |         |    X    |
| ProcTime        |   X   |    X   |    X    |         |    X    |
| Swap            |   X   |    X   |         |    X    |    X    |
| Uptime          |   X   |    X   |         |    X    |    X    |

## OS Specific Notes

### FreeBSD

Mount both `linprocfs` and `procfs` for compatability. Consider adding these
mounts to your `/etc/fstab` file so they are mounted automatically at boot.

```
sudo mount -t procfs proc /proc
sudo mkdir -p /compat/linux/proc
sudo mount -t linprocfs /dev/null /compat/linux/proc
```

## License

Apache 2.0
