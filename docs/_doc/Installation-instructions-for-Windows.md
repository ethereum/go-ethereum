---
title: Installation instructions for Windows
---
# Binaries

## Download stable binaries

All versions of Geth are built and available for download at https://geth.ethereum.org/downloads/.

The download page provides an installer as well as a zip file. The installer puts geth into your
PATH automatically. The zip file contains the command .exe files and can be used without installing.

1. Download zip file
1. Extract geth.exe from zip
1. Open a command prompt
1. chdir <path to geth.exe>
1. open geth.exe

# Source

## Compiling geth with tools from chocolatey

The Chocolatey package manager provides an easy way to get
the required build tools installed. If you don't have chocolatey yet,
follow the instructions on https://chocolatey.org to install it first.

Then open an Administrator command prompt and install the build tools
we need:

```text
C:\Windows\system32> choco install git
C:\Windows\system32> choco install golang
C:\Windows\system32> choco install mingw
``` 

Installing these packages will set up the `Path` environment variable.
Open a new command prompt to get the new `Path`. The following steps don't
need Administrator privileges.

Please ensure that the installed Go version is 1.7 (or any later version).

First we'll create and set up a Go workspace directory layout,
then clone the source.

***OBS*** If, during the commands below, you get the following message: 
```
 WARNING: The data being saved is truncated to 1024 characters.
```
Then that means that the `setx` command will fail, and proceeding will truncate the `Path`/`GOPATH`. If this happens, it's better to abort, and try to make some more room in `Path` before trying again. 

```text
C:\Users\xxx> set "GOPATH=%USERPROFILE%"
C:\Users\xxx> set "Path=%USERPROFILE%\bin;%Path%"
C:\Users\xxx> setx GOPATH "%GOPATH%"
C:\Users\xxx> setx Path "%Path%"
C:\Users\xxx> mkdir src\github.com\ethereum
C:\Users\xxx> git clone https://github.com/ethereum/go-ethereum src\github.com\ethereum\go-ethereum
C:\Users\xxx> cd src\github.com\ethereum\go-ethereum
C:\Users\xxx> go get -u -v golang.org/x/net/context
```

Finally, the command to compile geth is:

```text
C:\Users\xxx\src\github.com\ethereum\go-ethereum> go install -v ./cmd/...
```