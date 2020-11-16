## Vulnerability disclosures

In the software world, it is expected for security vulnerabilities to be immediately announced, thus giving operators an opportunity to take protective measure against attackers. 

Vulnerabilies can typically take two forms: 
1. Bugs that, if exploited, would harm the software operator. In the case of go-ethereum, examples of such bugs would be:
    - A bug that would allow remote reading or writing of OS files, or 
    - Remote command execution, or
    - Bugs that would leak cryptographic keys  
2. Bugs that, if exploited, would harm the Ethereum mainnet. In the case of go-ethereum, such bugs would typically be: 
    - Consensus vulnerabilities, which would cause a chain split, 
    - Denial-of-service during block processing, whereby a malicious transaction could cause the geth-portion of the network to crash.  
    - Denial-of-service via p2p networking, whereby portions of the network could be made inaccessible dur to crashes or resource consumption.

Historically, vulnerabilities in `geth` predominantly been of the second type, where the health of the network is a concern, rather than individual node operators. 

For bugs in category `2` above, we reserve the right to silently patch and ship fixes in new releases. 

### Why silent patches

In the case of Ethereum, it takes a lot of time (weeks, months) to get node operators to update even to a scheduled hard fork. 
If we were to highlight that a release contains important consensus or DoS fixes, there is always a risk of someone trying to beat updaters 
to the punch line and exploit the vulnerability. We do not strive for "security via obscurity", but delaying a potential attack 
sufficiently to make the majority of node operators immune may be worth the temporary "hit" to transparency.

The primary goal for the Geth team is the health of the Ethereum network as a whole.

The decision whether or not to publish details about a serious bug boils down to what the 
fallout would be in both cases and picking the one where the damage is smaller. 

At certain times, it's better to remain silent as shown by other projects 
too such as [Monero](https://www.getmonero.org/2017/05/17/disclosure-of-a-major-bug-in-cryptonote-based-currencies.html), 
[ZCash](https://electriccoin.co/blog/zcash-counterfeiting-vulnerability-successfully-remediated/) and 
[Bitcoin](https://www.coindesk.com/the-latest-bitcoin-bug-was-so-bad-developers-kept-its-full-details-a-secret).

### Public transparency

As of November 2020, our policy going forward is: 

- If we silently fix and ship a vulnerability in release `X`, then, 
- After 4-8 weeks, we will disclose that `X` contained a security-fix. 
- After an additional 4-8 weeks, we will publish the details about the vulnerability.

We hope that this provides sufficient balance between transparency versus the need for secrecy, and aids node operators and downstream projects
 in keeping up to date with what versions to run on their infrastructure.

In keeping with this policy, we have taken inspiration from [Solidity bug disclosure](https://solidity.readthedocs.io/en/develop/bugs.html) - see below.

## Disclosed vulnerabilities

In this folder, you can find a JSON-formatted list of some of the known security-relevant vulnerabilities concerning `geth`. 

The file itself is hosted in the Github repository, on the `gh-pages`-branch. 
The list was started in November 2020, and covers mainly `v1.9.7` and forward.

The JSON file of known bugs below is a list of objects, one for each bug, with the following keys:

- `name` 
  - Unique name given to the bug.
- `uid` 
  - Unique identifier of the bug. Format `GETH-<year>-<sequential id>`
- `summary`
    - Short description of the bug.
- `description`
    - Detailed description of the bug.
- `links`
    - List of releavnt URLs with more detailed information (optional).
- `introduced`
    - The first published compiler version that contained the bug (optional).
- `fixed`
    - The first published compiler version that did not contain the bug anymore.
- `published`
    - The date at which the bug became known publicly (optional).
- `severity`
    - Severity of the bug: `low`, `medium`, `high`, `critical`. 
    - Takes into account the severity of impact and likelihood of exploitation.
- `check`
    - This field contains a regular expression, which can be used against the reported `web3_clientVersion` of a node. If the check 
    matches, the node is with a high likelyhood affected by the vulnerability.