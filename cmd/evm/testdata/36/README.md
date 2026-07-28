## EIP-8297 binary tree state transition, with transactions

Four transactions on the `BinaryTree` fork, chosen so that one run covers every
way the post-state dump can go wrong. The tree cannot be walked back to
addresses - its leaves are keyed by a hash of the address and it keeps no
preimages - so the dump is rebuilt from the input allocation's keys plus the
ones the transition touched. Each transaction here attacks a different part of
that reconstruction:

- a **value transfer** to an address that does not exist yet, so an account
  **created** during the transition has to appear even though nothing in the
  input mentions it;
- a **call** into a contract that writes slot `0x02` (which lives in the
  account's header stem) and slot `0x400` (which lives in the overflow storage
  bucket), and clears slot `0x03`. The first two must be updated in **both
  storage homes**; the third must **disappear**, because zero is absence in the
  tree rather than a stored zero. Slot `0x05` is never touched and must
  **survive**, which only happens if the input allocation's keys are carried
  forward rather than just the mutated ones;
- a **contract creation** whose 40-byte runtime code spans more than one code
  chunk, exercising **code chunking** on an address that has to be discovered;
- a **create-and-selfdestruct** in a single transaction, which EIP-6780 deletes.
  The account is touched but must **not** appear in the dump.

Sibling fixture `../35` covers the same tree shapes with no transactions.

### What is verified against the reference, and what is not

The post-state was cross-checked against the execution specs reference
(`execution-specs`, branch `eip-8297-tests`) by replaying the same signed
transactions through its `binary_tree` fork. Comparing field by field, the two
agree on **every address, nonce, code body and storage slot** - including slot
`0x05` surviving, slot `0x03` being absent, and the self-destructed contract
being absent from both.

Exactly one field differs: the sender's balance, because the two projects do not
define this fork's gas the same way. The reference's `binary_tree` prices like
Amsterdam - its gas for a transaction is identical to its Amsterdam gas -
whereas geth composes EIP-4762 witness pricing onto the fork. For a plain value
transfer that is 183,600 gas there against 21,000 here. Both tools agree on
Osaka (27,900 each), so this is a disagreement about what the fork *is*, not
about either tool's accounting.

So the state root and the sender's balance here are geth's own, and everything
the disagreement does not touch is spec-blessed. Fixture `../35` carries no
transactions, which is why its root *can* be checked against the reference.

Reproducing the cross-check needs two things the reference requires and geth
does not: transactions pre-signed with `v`/`r`/`s` rather than a `secretKey`,
and a `blockHashes` entry in the environment.
