# Bor Overview
Bor is the Official Golang implementation of the Matic protocol. It is a fork of Go Ethereum - https://github.com/ethereum/go-ethereum and EVM compatible.

![Forks](https://img.shields.io/github/forks/maticnetwork/bor?style=social)
![Stars](https://img.shields.io/github/stars/maticnetwork/bor?style=social)
![Languages](https://img.shields.io/github/languages/count/maticnetwork/bor)
![Issues](https://img.shields.io/github/issues/maticnetwork/bor)
![PRs](https://img.shields.io/github/issues-pr-raw/maticnetwork/bor)
![MIT License](https://img.shields.io/github/license/maticnetwork/bor)
![contributors](https://img.shields.io/github/contributors-anon/maticnetwork/bor)
![size](https://img.shields.io/github/languages/code-size/maticnetwork/bor)
![lines](https://img.shields.io/tokei/lines/github/maticnetwork/bor)
[![Discord](https://img.shields.io/discord/714888181740339261?color=1C1CE1&label=Polygon%20%7C%20Discord%20%F0%9F%91%8B%20&style=flat-square)](https://discord.gg/zdwkdvMNY2)
[![Twitter Follow](https://img.shields.io/twitter/follow/0xPolygon.svg?style=social)](https://twitter.com/0xPolygon)

## How to contribute

### Contribution  Guidelines
We believe one of the things that makes Polygon special is its coherent design and we seek to retain this defining characteristic. From the outset we defined some guidelines to ensure new contributions only ever enhance the project:

* Quality: Code in the Polygon project should meet the style guidelines, with sufficient test-cases, descriptive commit messages, evidence that the contribution does not break any compatibility commitments or cause adverse feature interactions, and evidence of high-quality peer-review
* Size: The Polygon project’s culture is one of small pull-requests, regularly submitted. The larger a pull-request, the more likely it is that you will be asked to resubmit as a series of self-contained and individually reviewable smaller PRs
* Maintainability: If the feature will require ongoing maintenance (eg support for a particular brand of database), we may ask you to accept responsibility for maintaining this feature
### Submit an issue

- Create a [new issue](https://github.com/maticnetwork/bor/issues/new/choose)
- Comment on the issue (if you'd like to be assigned to it) - that way [our team can assign the issue to you](https://github.blog/2019-06-25-assign-issues-to-issue-commenters/).
- If you do not have a specific contribution in mind, you can also browse the issues labelled as `help wanted`
- Issues that additionally have the `good first issue` label are considered ideal for first-timers

### Fork the repository (repo)

- If you're not sure, here's how to [fork the repo](https://help.github.com/en/articles/fork-a-repo)

- If this is your first time forking our repo, this is all you need to do for this step:

    ```
    $ git clone git@github.com:[your_github_handle]/bor
    ```

- If you've already forked the repo, you'll want to ensure your fork is configured and that it's up to date. This will save you the headache of potential merge conflicts.

- To [configure your fork](https://docs.github.com/en/github/collaborating-with-issues-and-pull-requests/configuring-a-remote-for-a-fork):

    ```
    $ git remote add upstream https://github.com/maticnetwork/bor
    ```

- To [sync your fork with the latest changes](https://docs.github.com/en/github/collaborating-with-issues-and-pull-requests/syncing-a-fork):

    ```
    $ git checkout master
    $ git fetch upstream
    $ git merge upstream/master
    ```

### Building the source

- Building `bor` requires both a Go (version 1.19 or later) and a C compiler. You can install
them using your favourite package manager. Once the dependencies are installed, run

     ```shell
     $ make bor
     ```

### Make awesome changes!

1. Create new branch for your changes

    ```
    $ git checkout -b new_branch_name
    ```

2. Commit and prepare for pull request (PR). In your PR commit message, reference the issue it resolves (see [how to link a commit message to an issue using a keyword](https://docs.github.com/en/free-pro-team@latest/github/managing-your-work-on-github/linking-a-pull-request-to-an-issue#linking-a-pull-request-to-an-issue-using-a-keyword).


    Checkout our [Git-Rules](https://wiki.polygon.technology/docs/contribute/orientation/#git-rules)

    ```
    $ git commit -m "brief description of changes [Fixes #1234]"
    ```

3. Push to your GitHub account

    ```
    $ git push
    ```

### Submit your PR

- After your changes are commited to your GitHub fork, submit a pull request (PR) to the `master` branch of the `maticnetwork/bor` repo
- In your PR description, reference the issue it resolves (see [linking a pull request to an issue using a keyword](https://docs.github.com/en/free-pro-team@latest/github/managing-your-work-on-github/linking-a-pull-request-to-an-issue#linking-a-pull-request-to-an-issue-using-a-keyword))
  - ex. `Updates out of date content [Fixes #1234]`
- Why not say hi and draw attention to your PR in [our discord server](https://discord.gg/zdwkdvMNY2)?

### Wait for review

- The team reviews every PR
- Acceptable PRs will be approved & merged into the `master` branch

<hr style="margin-top: 3em; margin-bottom: 3em;">

## Release

- You can [view the history of releases](https://github.com/maticnetwork/bor/releases), which include PR highlights

<hr style="margin-top: 3em; margin-bottom: 3em;">


## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html),
also included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also
included in our repository in the `COPYING` file.

<hr style="margin-top: 3em; margin-bottom: 3em;">

## Join our Discord server

Join Polygon community  – share your ideas or just say hi over [on Discord](https://discord.gg/zdwkdvMNY2).
