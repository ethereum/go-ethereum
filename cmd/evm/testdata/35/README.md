## EIP-8297 binary tree state transition

A transition on the `BinaryTree` fork: the EIP-8297 partitioned binary tree
carrying an account whose storage spans both of its homes - slot 2 lives in
the account's header stem, slot 0x400 in the overflow storage bucket - along
with chunked code. The code is what the account header does *not* hold:
every chunk lives in the content-addressed code zone, keyed by the code
hash, so the header stem here carries only basic data, the code hash and
slot 2.

The expected state root was verified independently against the execution
specs reference implementation (`execution-specs` at `1d47f584d`, which
moved all code chunks into the code zone), by embedding the same allocation
through `ethereum.binary_trie` and hashing it: both produce

    0xf76bce2471940ba9c60fe5fcd8e0cac6c6831d2402d5494a4f535aa210a78c70

so this fixture pins the key derivation, the code chunking and the tree
hashing against the spec rather than against geth's own output.

Geth's own execution is the second witness: `TestT8n` runs this fixture and
must reproduce the reference-derived root exactly.

The expected allocation is spec-blessed too: the reference emits the same
account, code and both storage slots. Carrying no transactions is what makes
that possible - see `../36`, which does run transactions and therefore can
only match the reference on everything except gas.
