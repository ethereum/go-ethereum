### Windows 

Download and run the installer found at http://golang.org/doc/install

### OS X

Download an install the darwin binary from https://golang.org/dl/

You can also install go using the Homebrew package manager.

### Linux

#### Ubuntu

The Ubuntu repositories carry an old version of Go.

Ubuntu users can use the 'gophers' PPA to install an up to date version of Go (version 1.7 or later is preferred).
See https://launchpad.net/~gophers/+archive/ubuntu/archive for more information.
Note that this PPA requires adding `/usr/lib/go-1.X/bin` to the executable PATH.

#### Other distros

Download the latest distribution

`curl -O https://storage.googleapis.com/golang/go1.7.3.linux-amd64.tar.gz`

Unpack it to the `/usr/local` (might require sudo)

`tar -C /usr/local -xzf go1.7.3.linux-amd64.tar.gz`

#### Set GOPATH and PATH

For Go to work properly, you need to set the following two environment variables:

- Setup a go folder `mkdir -p ~/go; echo "export GOPATH=$HOME/go" >> ~/.bashrc` 
- Update your path `echo "export PATH=$PATH:$HOME/go/bin:/usr/local/go/bin" >> ~/.bashrc`
- Read the environment variables into current session: `source ~/.bashrc`