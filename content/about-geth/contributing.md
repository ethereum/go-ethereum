---
title: Contributing
---

We welcome contributions from anyone on the internet, and are grateful for even the smallest of fixes!

## Contributing to the Geth source code

If you'd like to contribute to the Geth source code, please fork the 
[Github repository](https://github.com/ethereum/go-ethereum), fix, commit and send a pull request for the 
maintainers to review and merge into the main code base. If you wish to submit more complex changes 
though, please check up with the core devs first on our Discord Server to ensure those changes are in 
line with the general philosophy of the project and/or get some early feedback which can make both your 
efforts much lighter as well as our review and merge procedures quick and simple.

Please make sure your contributions adhere to our coding guidelines:

* Code must adhere to the official Go formatting guidelines (i.e. uses gofmt).
* Code must be documented adhering to the official Go commentary guidelines.
* Pull requests need to be based on and opened against the master branch.
* Commit messages should be prefixed with the package(s) they modify.
	E.g. "eth, rpc: make trace configs optional"


## Contributing to the Geth website

The Geth website is hosted separately from Geth itself. The contribution guidelines are the same. Please
for the Geth website Github repository and raise pull requests for the maintainers to review and merge.

## License

The go-ethereum library (i.e. all code outside of the cmd directory) is licensed under the GNU Lesser General Public License v3.0, also included in our repository in the COPYING.LESSER file.

The go-ethereum binaries (i.e. all code inside of the cmd directory) is licensed under the GNU General Public License v3.0, also included in our repository in the COPYING file.
