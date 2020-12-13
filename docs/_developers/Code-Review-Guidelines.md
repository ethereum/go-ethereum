---
title: Code Review Guidelines
sort_key: B
---

The only way to get code into go-ethereum is to send a pull request. Those pull requests
need to be reviewed by someone. This document is a guide that explains our expectations
around PRs for both authors and reviewers.

## Terminology

* The **author** of a pull request is the entity who wrote the diff and submitted it to
  GitHub.
* The **team** consists of people with commit rights on the go-ethereum repository.
* The **reviewer** is the person assigned to review the diff. The reviewer must be a team
  member.
* The **code owner** is the person responsible for the subsystem being modified by the PR.

## The Process

The first decision to make for any PR is whether it's worth including at all. This
decision lies primarily with the code owner, but may be negotiated with team members.

To make the decision we must understand what the PR is about. If there isn't enough
description content or the diff is too large, request an explanation. Anyone can do this
part.

We expect that reviewers check the style and functionality of the PR, providing comments
to the author using the GitHub review system. Reviewers should follow up with the PR until
it is in good shape, then **approve** the PR. Approved PRs can be merged by any code owner.

When communicating with authors, be polite and respectful.

### Code Style

We expect `gofmt`ed code. For contributions of significant size, we expect authors to
understand and use the guidelines in [Effective Go][effgo]. Authors should avoid common
mistakes explained in the [Go Code Review Comments][revcomment] page.

### Functional Checks

For PRs that fix an issue, reviewers should try reproduce the issue and verify that the
pull request actually fixes it. Authors can help with this by including a unit test that
fails without (and passes with) the change.

For PRs adding new features, reviewers should attempt to use the feature and comment on
how it feels to use it. Example: if a PR adds a new command line flag, use the program
with the flag and comment on whether the flag feels useful.

We expect appropriate unit test coverage. Reviewers should verify that new code is covered
by unit tests.

### CI

Code submitted must pass all unit tests and static analysis ("lint") checks. We use Travis
CI to test code on Linux, macOS and AppVeyor to test code on Microsoft Windows.

For failing CI builds, the issue may not be related to the PR itself. Such failures are
usually related to flakey tests. These failures can be ignored (authors don't need to fix
unrelated issues), but please file a GH issue so the test gets fixed eventually.

### Commit Messages

Commit messages on the master branch should follow the rule below. PR authors are not
required to use any particular style because the message can be modified at merge time.
Enforcing commit message style is the responsibility of the person merging the PR.

The commit message style we use is similar to the style used by the Go project:

The first line of the change description is conventionally a one-line summary of the
change, prefixed by the primary affected Go package. It should complete the sentence "This
change modifies go-ethereum to _____." The rest of the description elaborates and should
provide context for the change and explain what it does.

Template:

```text
package/path: change XYZ
 
Longer explanation of the change in the commit. You can use
multiple sentences here. It's usually best to include content
from the PR description in the final commit message.
 
issue notices, e.g. "Fixes #42353".
```

### Special Situations And How To Deal With Them

As a reviewer, you may find yourself in one of the sitations below. Here's how to deal
with those:

* The author doesn't follow up: ping them after a while (i.e. after a few days). If there
  is no further response, close the PR or complete the work yourself.

* Author insists on including refactoring changes alongside bug fix: We can tolerate small
  refactorings alongside any change. If you feel lost in the diff, ask the author to
  submit the refactoring as an independent PR, or at least as an independent commit in the
  same PR.

* Author keeps rejecting your feedback: reviewers have authority to reject any change for technical reasons. If you're unsure, ask the team for a second opinion. You may close the PR if no consensus can be reached. 

[effgo]: https://golang.org/doc/effective_go.html
[revcomment]: https://github.com/golang/go/wiki/CodeReviewComments
