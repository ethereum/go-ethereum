---
title: Vulnerability disclosure
sort_key: A
---

## About disclosures

In the software world, it is expected for security vulnerabilities to be immediately
announced, thus giving operators an opportunity to take protective measure against
attackers.

Vulnerabilies typically take two forms:

1. Vulnerabilies that, if exploited, would harm the software operator. In the case of
   go-ethereum, examples would be:
    - A bug that would allow remote reading or writing of OS files, or
    - Remote command execution, or
    - Bugs that would leak cryptographic keys
2. Vulnerabilies that, if exploited, would harm the Ethereum mainnet. In the case of
   go-ethereum, examples would be:
    - Consensus vulnerabilities, which would cause a chain split,
    - Denial-of-service during block processing, whereby a malicious transaction could cause the geth-portion of the network to crash.
    - Denial-of-service via p2p networking, whereby portions of the network could be made
      inaccessible due to crashes or resource consumption.

In most cases so far, vulnerabilities in `geth` have been of the second type, where the
health of the network is a concern, rather than individual node operators. For such
issues, we reserve the right to silently patch and ship fixes in new releases.

### Why silent patches

In the case of Ethereum, it takes a lot of time (weeks, months) to get node operators to
update even to a scheduled hard fork. If we were to highlight that a release contains
important consensus or DoS fixes, there is always a risk of someone trying to beat node
operators to the punch, and exploit the vulnerability. Delaying a potential attack
sufficiently to make the majority of node operators immune may be worth the temporary loss
of transparency.

The primary goal for the Geth team is the health of the Ethereum network as a whole, and
the decision whether or not to publish details about a serious vulnerability boils down to
minimizing the risk and/or impact of discovery and exploitation.

At certain times, it's better to remain silent. This practice is also followed by other
projects such as
[Monero](https://www.getmonero.org/2017/05/17/disclosure-of-a-major-bug-in-cryptonote-based-currencies.html),
[ZCash](https://electriccoin.co/blog/zcash-counterfeiting-vulnerability-successfully-remediated/)
and
[Bitcoin](https://www.coindesk.com/the-latest-bitcoin-bug-was-so-bad-developers-kept-its-full-details-a-secret).

### Public transparency

As of November 2020, our policy going forward is:

- If we silently fix a vulnerability and include the fix in release `X`, then,
- After 4-8 weeks, we will disclose that `X` contained a security-fix.
- After an additional 4-8 weeks, we will publish the details about the vulnerability.

We hope that this provides sufficient balance between transparency versus the need for
secrecy, and aids node operators and downstream projects in keeping up to date with what
versions to run on their infrastructure.

In keeping with this policy, we have taken inspiration from [Solidity bug disclosure](https://solidity.readthedocs.io/en/develop/bugs.html) - see below.

## Disclosed vulnerabilities

In this folder, you can find a JSON-formatted list
([`vulnerabilities.json`](vulnerabilities.json)) of some of the known security-relevant
vulnerabilities concerning `geth`.

As of `geth` version `1.9.25`, geth has a built-in command to check whether it is affected
by any publically disclosed vulnerability, using the command `geth version-check`. This
command will fetch the latest json file (and the accompanying
[signature-file](vulnerabilities.json.minisig), and cross-check the data against it's own
version number.

The file itself is hosted in the Github repository, on the `gh-pages`-branch. The list was
started in November 2020, and covers mainly `v1.9.7` and forward.

The JSON file of known vulnerabilities below is a list of objects, one for each
vulnerability, with the following keys:

- `name`
  - Unique name given to the vulnerability.
- `uid`
  - Unique identifier of the vulnerability. Format `GETH-<year>-<sequential id>`
- `summary`
  - Short description of the vulnerability.
- `description`
  - Detailed description of the vulnerability.
- `links`
  - List of relevant URLs with more detailed information (optional).
- `introduced`
  - The first published Geth version that contained the vulnerability (optional).
- `fixed`
  - The first published Geth version that did not contain the vulnerability anymore.
- `published`
  - The date at which the vulnerability became known publicly (optional).
- `severity`
  - Severity of the vulnerability: `low`, `medium`, `high`, `critical`.
  - Takes into account the severity of impact and likelihood of exploitation.
- `check`
  - This field contains a regular expression, which can be used against the reported `web3_clientVersion` of a node. If the check
    matches, the node is with a high likelyhood affected by the vulnerability.
- `CVE`
  - The assigned `CVE` identifier, if available (optional)

### What about Github security advisories

We prefer to not rely on Github as the only/primary publishing protocol for security
advisories, but we plan to use the Github-advisory process as a second channel for
disseminating vulnerability-information.

Advisories published via Github can be accessed [here](https://github.com/ethereum/go-ethereum/security/advisories?state=published).
