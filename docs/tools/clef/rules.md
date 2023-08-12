---
title: Rules
description: Introduction to automated rulesets in Clef
---

Rules in Clef are sets of conditions that determine whether a given action can be approved automatically without requiring manual intervention from the user. This can be useful for automatically approving transactions between a user's own accounts, or approving patterns that are commonly used by applications. Automatic signing also requires Clef to have access to account passwords which is configured independently of the ruleset.

Rules can define arbitrary conditions such as:

- Auto-approve 10 transactions with contract `CasinoDapp`, with value between `0.05 ether` and `1 ether` per 24h period.

- Auto-approve transactions to contract `Uniswapv2` with `value` up to 1 ether, if `gas < 44k` and `gasPrice < 40Gwei`.

- Auto-approve signing if the data to be signed contains the string `"approve_me"`.

- Auto-approve any requests to list accounts in keystore if the request arrives over IPC

Because the rules are Javascript files they can be customized to implement any arbitrary logic on the available request data.

This page will explain how rules are implemented in Clef and how best to manage credentials when automatic rulesets are enabled.

## Rule Implementation {#rule-implementation}

The ruleset engine acts as a gatekeeper to the command line interface - it auto-approves any requests that meet the conditions defined in a set of authenticated rule files. This prevents the user from having to manually approve or reject every request - instead they can define common patterns in a rule file and abstract that task away to the ruleset engine. The general architecture is as follows:

![Clef ruleset logic](/images/docs/clef_ruleset.png)

When Clef receives a request, the ruleset engine evaluates a Javascript file for each method defined in the internal [UI API docs](/docs/tools/clef/apis). For example the code snippet below is an example ruleset that calls the function `ApproveTx`. The call to `ApproveTx` is invoking the `ui_approveTx` [JSON_RPC API endpoint](/docs/tools/clef/apis). Every time an RPC method is invoked the Javascript code is executed in a freshly instantiated virtual machine.

```js
function asBig(str) {
	if (str.slice(0, 2) == "0x") {
		return new BigNumber(str.slice(2), 16)
	}
	return new BigNumber(str)
}

// Approve transactions to a certain contract if value is below a certain limit
function ApproveTx(req) {
	var limit = big.Newint("0xb1a2bc2ec50000")
	var value = asBig(req.transaction.value);

	if (req.transaction.to.toLowerCase() == "0xae967917c465db8578ca9024c205720b1a3651a9" && value.lt(limit)) {
		return "Approve"
	}
	// If we return "Reject", it will be rejected.
	// By not returning anything, the decision to approve/reject
	// will be passed to the next UI, for manual processing
}

// Approve listings if request made from IPC
function ApproveListing(req){
    if (req.metadata.scheme == "ipc"){ return "Approve"}
}
```

When a request is made via the external API, the logic flow is as follows:

- Request is made to the `signer` binary using external API
- `signer` calls the UI - in this case the ruleset engine

- UI evaluates whether the call conforms to rules in an attested rulefile

- Assuming the call returns "Approve", request is signed.

There are three possible outcomes from the ruleset engine that are handled in different ways:

| Return value  | Action                                  |
| ------------- | --------------------------------------- |
| "Approve"     | Auto-approve request                    |
| "Reject"      | Auto-reject request                     |
| Anything else | Pass decision to UI for manual approval |

There are some additional noteworthy implementation details that are important for defining rules correctly in `ruleset.js`:

- The code in `ruleset.js` **cannot** load external Javascript files.
- The Javascript engine can access `storage` and `console`
- The only preloaded library in the Javascript environment is `bignumber.js` version `2.0.3`.
- Each invocation is made in a fresh virtual machine meaning data cannot be stored in global variables between invocations.
- Since no global variable storage is available, disk backed `storage` must be used - rules should not rely on ephemeral data.
- Javascript API parameters are always objects. This ensures parameters are accessed by _key_ to avoid misordering errors.
- Otto VM uses ES5. ES6-specific features (such as Typed Arrays) are not supported.
- The regular expression engine (re2/regexp) in Otto VM is not fully compatible with the [ECMA5 specification](https://tc39.es/ecma262/#sec-intro).
- [Strict mode](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Strict_mode) is not supported. "use strict" will parse but it does nothing.

## Credential management {#credential-management}

The ability to auto-approve transaction requires that the signer has the necessary credentials, i.e. account passwords, to decrypt keyfiles. These are stored encrypted as follows:

When the `signer` is started it generates a seed that is locked with a user specified password. The seed is saved to a location that defaults to `$HOME/.clef/masterseed.json`. The `seed` itself is a blob of bytes.

The `signer` uses the `seed` to:

- Generate the `path` where the configuration and credentials data are stored.
  - `$HOME/.clef/790046d38025/config.json`
  - `$HOME/.clef/790046d38025/credentials.json`
- Generate the encryption password for the config and credentials files.

`config.json` stores the hashes of any attested rulesets. `credentials.json` stores encrypted account passwords. The masterseed is required to decrypt these files. The decrypted account passwords can then be used to decrypt keyfiles.

## Security {#security}

### The Javascript VM {#javascript-vm}

The downside of the very flexible rule implementation included in Clef is that the `signer` binary needs to contain a Javascript engine. This is an additional attack surface. The only viable attack is for an adversary to somehow extract cryptographic keys from memory during the Javascript VM execution. The hash-based rule attestation condition means the actual Javascript code executed by the Javascript engine is not a viable attack surface -- since if the attacker can control the ruleset, a much simpler attack would be to surreptitiously insert an attested "always-approve" rule instead of attempting to exploit the Javascript virtual machine. The Javascript engine is quite simple to implement and there are currently no known security vulnerabilities, not have there been any security problems identified for the similar Javascript VM implemented in Geth.

### Writing rules {#writing-rules}

Since the user has complete freedom to write custom rules, it is plausible that those rules could create unintended security vulnerabilities. This can only really be protected by coding very carefully and trying to test rulesets (e.g. on a private testnet) before implementing them on a public network.

Javascript is very flexible but also easy to write incorrectly. For example, users might assume that javascript can handle large integers natively rather than explicitly using `bigInt`. This is an error commonly encountered in the Ethereum context when users attempt to multiply `gas` by `gasCost`.

It’s unclear whether any other language would be more secure - there is always the possibility of implementing an insecure rule.

### Credential security {#credential-security}

Clef implements a secure, encrypted vault for storing sensitive data. This vault is encrypted using a `masterseed` which the user is responsible for storing and backing up safely and securely. Since this `masterseed` is used to decrypt the secure vault, and its security is not handled by Clef, it could represent a security vulnerability if the user does not implement best practise in keeping it safe.

The same is also true for keys. Keys are not stored by Clef, they are only accessed using account passwords that Clef does store in its vault. The keys themselves are stored in an external `keystore` whose security is the responsibility of the user. If the keys are compromised, the account is not safe irrespective of the security benefits derived from Clef.

## Ruleset examples {#ruleset-examples}

Below are some examples of `ruleset.js` files.

### Example 1: Allow destination

```js
function ApproveTx(r) {
  if (r.transaction.to.toLowerCase() == '0x0000000000000000000000000000000000001337') {
    return 'Approve';
  }
  if (r.transaction.to.toLowerCase() == '0x000000000000000000000000000000000000dead') {
    return 'Reject';
  }
  // Otherwise goes to manual processing
}
```

### Example 2: Allow listing

```js
function ApproveListing() {
  return 'Approve';
}
```

### Example 3: Approve signing data

```js
function ApproveSignData(req) {
  if (req.address.toLowerCase() == '0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3') {
    if (req.messages[0].value.indexOf('bazonk') >= 0) {
      return 'Approve';
    }
    return 'Reject';
  }
  // Otherwise goes to manual processing
}
```

### Example 4: Rate-limited window

```js
function big(str) {
  if (str.slice(0, 2) == '0x') {
    return new BigNumber(str.slice(2), 16);
  }
  return new BigNumber(str);
}

// Time window: 1 week
var window = 1000 * 3600 * 24 * 7;

// Limit : 1 ether
var limit = new BigNumber('1e18');

function isLimitOk(transaction) {
  var value = big(transaction.value);
  // Start of our window function
  var windowstart = new Date().getTime() - window;

  var txs = [];
  var stored = storage.get('txs');

  if (stored != '') {
    txs = JSON.parse(stored);
  }
  // First, remove all that have passed out of the time-window
  var newtxs = txs.filter(function (tx) {
    return tx.tstamp > windowstart;
  });
  console.log(txs, newtxs.length);

  // Secondly, aggregate the current sum
  sum = new BigNumber(0);

  sum = newtxs.reduce(function (agg, tx) {
    return big(tx.value).plus(agg);
  }, sum);
  console.log('ApproveTx > Sum so far', sum);
  console.log('ApproveTx > Requested', value.toNumber());

  // Would we exceed weekly limit ?
  return sum.plus(value).lt(limit);
}
function ApproveTx(r) {
  if (isLimitOk(r.transaction)) {
    return 'Approve';
  }
  return 'Nope';
}

/**
 * OnApprovedTx(str) is called when a transaction has been approved and signed. The parameter
 * 'response_str' contains the return value that will be sent to the external caller.
 * The return value from this method is ignore - the reason for having this callback is to allow the
 * ruleset to keep track of approved transactions.
 *
 * When implementing rate-limited rules, this callback should be used.
 * If a rule responds with neither 'Approve' nor 'Reject' - the tx goes to manual processing. If the user
 * then accepts the transaction, this method will be called.
 *
 * TLDR; Use this method to keep track of signed transactions, instead of using the data in ApproveTx.
 */
function OnApprovedTx(resp) {
  var value = big(resp.tx.value);
  var txs = [];
  // Load stored transactions
  var stored = storage.get('txs');
  if (stored != '') {
    txs = JSON.parse(stored);
  }
  // Add this to the storage
  txs.push({ tstamp: new Date().getTime(), value: value });
  storage.put('txs', JSON.stringify(txs));
}
```

## Summary {#summary}

Rules are sets of conditions encoded in Javascript files that enable certain actions to be auto-approved by Clef. This page outlined the implementation details and security considerations that will help to build suitable ruleset files. See the [Clef GitHub](https://github.com/ethereum/go-ethereum/tree/master/cmd/clef) for further reading.
