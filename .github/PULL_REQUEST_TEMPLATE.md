Thank you for helping us improve Go Ethereum!

Before contributing, please read
[How to contribute](https://github.com/ethereum/go-ethereum#contribution) and the
[Code Review Guidelines](https://geth.ethereum.org/docs/developers/code-review-guidelines).
It's also recommended to open an issue first where you can discuss the issue
and your proposed solution with the code owners.

# Checklist

- For PRs that fix an issue, reviewers should try reproduce the issue and verify that the pull request actually fixes it. Authors can help with this by including a unit test that fails without (and passes with) the change.
- For PRs adding new features, reviewers should attempt to use the feature and comment on how it feels to use it. Example: if a PR adds a new command line flag, use the program with the flag and comment on whether the flag feels useful.
- We expect appropriate unit test coverage. Reviewers should verify that new code is covered by unit tests.
- Code submitted must pass all unit tests and static analysis (“lint”) checks. We use Travis CI to test code on Linux, macOS and AppVeyor to test code on Microsoft Windows.
- For failing CI builds, the issue may not be related to the PR itself. Such failures are usually related to flaky tests. These failures can be ignored (authors don’t need to fix unrelated issues), but please file a GH issue so the test gets fixed eventually.

# Reference/Link to the issue solved with this PR (if any)

# Description of the problem
*Please be as precise as possible: what issue you experienced, how often...*

# Description of the solution
*How you solved the issue and the other things you considered and maybe rejected*
