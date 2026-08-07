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
  chunk, exercising **code chunking**. The chunks land in the code zone under
  the code hash rather than in the account, so the dump still has to discover
  the address by other means;
- a **create-and-selfdestruct** in a single transaction, which EIP-6780 deletes.
  The account is touched but must **not** appear in the dump.

Sibling fixture `../35` covers the same tree shapes with no transactions.

### What is verified against the reference, and what is not

The post-state was cross-checked against the execution specs reference
(`execution-specs` at `1d47f584d`) by replaying the same signed transactions
through its `binary_tree` fork. Comparing field by field, the two agree on
**every address, nonce, code body and storage slot** - including slot `0x05`
surviving, slot `0x03` being absent, and the self-destructed contract being
absent from both.

The root above is the reference's too, in a narrower sense than `../35`'s:
feeding the post-state dumped below back through the reference's embedding
reproduces it exactly. That pins the *embedding* - key derivation, chunk
placement, hashing - against the spec, while leaving the *execution* that
produced the post-state geth's own, for the reason below.

The two used to disagree on gas - this fork prices like Amsterdam in both
projects, but geth carried draft-era values for the EIP-8038/EIP-2780
repricing, so the sender's balance (and with it the root) was geth's own.
Those schedules are aligned to the merged EIPs now, and the whole EEST
binary-tree suite passes against geth, so the balances here are the gas both
projects agree on.

Both roots are now witnessed twice over: `TestT8n` executes the fixtures
through geth, and feeding the resulting post-state through the reference's
embedding reproduces the same root byte for byte. Fixture `../35` is stronger
still - it carries no transactions, so the reference derives its root from
the *input* allocation without executing anything.

Reproducing the replay cross-check needs two things the reference requires and
geth does not: transactions pre-signed with `v`/`r`/`s` rather than a
`secretKey`, and a `blockHashes` entry in the environment. Re-deriving the root
from the post-state needs neither - feed each account below through
`embed_account` and `embed_storage_slot`, then hash.
