## Go Ethereum

Golang execution layer implementation of the Ethereum protocol.

## Orakle

Orakle open-source EIP team repository.

## How to Work

### If fork

the case for working on personal repository after fork (recommended)

1. Setting for the first time

```
// add remote upstream
git remote add upstream https://github.com/orakle-opensource/EIP_opensource.git
```
2. Working on personal repository
```
// create branch to work
git checkout -b [branch_name]

// rebase upstream commits into branch
git rebase upstream/master -i

// staging and commit
git add .
git commit -m "message"

// push
git push -u origin [branch_name]

// go to upstream repo(https://github.com/orakle-opensource/EIP_opensource.git) and create pull request
```
3. After merging pull request
```
// checkout master and pull upstream commits
git checkout master
git pull upstream master
```

### If branch

the case for working by creating branch on our repository

```
// create branch -> commit -> push
git checkout -b [branch_name]
git add . | git commit -m "message"
git push -u origin [branch_name]

// go to repo and create pull request
```

## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html),
also included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) are licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also
included in our repository in the `COPYING` file.
