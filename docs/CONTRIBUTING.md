# Contributing

* [Reporting Bugs](#bugs)
* [General Procedure](#general_procedure)
  * [Development Procedure](#dev_procedure)
  * [Dependencies](#dependencies)
  * [Testing](#testing)
  * [Branching Model and Release](#braching_model_and_release)
  * [PR Targeting](#pr_targeting)
  * [Pull Requests](#pull_requests)
  * [Process for reviewing PRs](#reviewing_prs)
  * [Pull Merge Procedure](#pull_merge_procedure)


## <span id="bugs">Reporting Bugs</span>

Please file bugs in the [GitHub Issue
Tracker](https://github.com/covalenthq/bsp-geth). Include at
least the following:

 - What happened
 - What did you expect to happen instead of what *did* happen, if it's
   not crazy obvious
 - What operating system, operating system version and version of
   go-ethereum you are running
 - Console log entries, where possible and relevant

If you're not sure whether something is relevant, erring on the side of
too much information will never be a cause for concern.


## <span id="general_procedure">General Procedure</span>

Contributing to this repo can mean many things such as participating in discussion or proposing code changes. To ensure a smooth workflow for all contributors, the following general procedure for contributing has been established:

1. Either [open](https://github.com/covalenthq/bsp-geth/issues/new/choose)
   or [find](https://github.com/covalenthq/bsp-geth/issues) an issue you have identified and would like to contribute to
   resolving.

2. Participate in thoughtful discussion on that issue.

3. If you would like to contribute:
    1. If the issue is a proposal, ensure that the proposal has been accepted by the Covalent team.
    2. Ensure that nobody else has already begun working on the same issue. If someone already has, please make sure to contact the individual to collaborate.
    3. If nobody has been assigned the issue and you would like to work on it, make a comment on the issue to inform the
       community of your intentions to begin work. Ideally, wait for confirmation that no one has started it. However,
       if you are eager and do not get a prompt response, feel free to dive on in!
    4. Follow standard Github best practices:
        1. Fork the repo
        2. Branch from the HEAD of `main`(For core developers working within the go-ethereum repo, to ensure a clear ownership of branches, branches must be named with the convention `{moniker}/{issue#}-branch-name`).
        3. Make commits
        4. Submit a PR to `main`
    5. Be sure to submit the PR in `Draft` mode. Submit your PR early, even if it's incomplete as this indicates to the community you're working on something and allows them to provide comments early in the development process.
    6. When the code is complete it can be marked `Ready for Review`.
    7. Be sure to include a relevant change log entry in the `Unreleased` section of `CHANGELOG.md` (see file for log
       format).
    8. Please make sure to run `gofmt` before every commit - the easiest way to do this is having your editor run it for you upon saving a file. Additionally, please ensure that your code is lint compliant by running `make lint` . There are CI tests built into the bsp-geth repository and all PR’s will require that these tests pass before they are able to be merged.

**Note**: for very small or blatantly obvious problems (such as typos), it is not required to open an issue to submit a
PR, but be aware that for more complex problems/features, if a PR is opened before an adequate design discussion has
taken place in a github issue, that PR runs a high likelihood of being rejected.

Looking for a good place to start contributing? How about checking out
some [good first issues](https://github.com/covalenthq/bsp-geth/issues).

### <span id="dev_procedure">Development Procedure</span>

1. The latest state of development is on `main`.
2. `main` must never
   fail `make lint, make geth, make all `
3. No `--force` onto `main` (except when reverting a broken commit, which should seldom happen).
4. Create your feature branch from `main` either on `github.com/covalenthq/bsp-geth`, or your fork (
   using `git remote add origin`).
5. Before submitting a pull request, begin `git rebase` on top of `main`.
6. Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting)
   guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
7. Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary)
   guidelines.
8. Pull requests need to be based on and opened against the `main` branch.
9. Commit messages should be prefixed with the package(s) they modify.
   * E.g. "eth, rpc: make trace configs optional"


### <span id="dependencies">Dependencies</span>

We use [Go Modules](https://github.com/golang/go/wiki/Modules) to manage dependency versions.

The main branch of every bsp-geth repository should just build with `go get`, which means they should be kept up-to-date
with their dependencies, so we can get away with telling people they can just `go get` our software. Since some dependencies are not under our control, a third party may break our build, in which case we can fall back
on `go mod tidy -v`.

### <span id="testing">Testing</span>

Covalent uses [GitHub Actions](https://github.com/features/actions) for automated [integration testing](https://github.com/covalenthq/bsp-geth/actions).

### <span id="braching_model_and_release">Branching Model and Release</span>

User-facing repos should adhere to the [trunk based development branching model](https://trunkbaseddevelopment.com/).

Libraries need not follow the model strictly, but would be wise to.

bsp-geth utilizes [semantic versioning](https://semver.org/).

### <span id="pr_targeting">PR Targeting</span>

Ensure that you base and target your PR on the `main` branch.

All feature additions should be targeted against `main`. Bug fixes for an outstanding release candidate should be
targeted against the release candidate branch.

### <span id="pull_requests">Pull Requests</span>

To accommodate the review process, we suggest that PRs are categorically broken up. Ideally each PR addresses only a
single issue. Additionally, as much as possible code refactoring and cleanup should be submitted as separate PRs from
bug fixes/feature-additions.

### <span id="reviewing_prs">Process for reviewing PRs</span>

All PRs require at least 1 Review before merge. When reviewing PRs, please use the following review explanations:

1. `LGTM` without an explicit approval means that the changes look good, but you haven't pulled down the code, run tests
   locally and thoroughly reviewed it.
2. `Approval` through the GH UI means that you understand the code, documentation/spec is updated in the right places,
   you have pulled down and tested the code locally. In addition:
    * You must think through whether any added code could be partially combined (DRYed) with existing code.
    * You must think through any potential security issues or incentive-compatibility flaws introduced by the changes.
    * Naming convention must be consistent with the rest of the codebase.
    * Code must live in a reasonable location, considering dependency structures (e.g. not importing testing modules in
      production code, or including example code modules in production code).
    * If you approve of the PR, you are responsible for fixing any of the issues mentioned here.
3. If you are only making "surface level" reviews, submit any notes as `Comments` without adding a review.

### <span id="pull_merge_procedure">Pull Merge Procedure</span>

1. Ensure pull branch is rebased on `main`.
2. Ensure that all CI tests pass.
3. Squash merge pull request.