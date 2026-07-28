## EIP-8297 binary tree state transition

A transition on the `BinaryTree` fork: the EIP-8297 partitioned binary tree
carrying an account whose storage spans both of its homes - slot 2 lives in
the account's header stem, slot 0x400 in the overflow storage bucket - along
with chunked code.

The expected state root was verified independently against the execution
specs reference implementation (`execution-specs`, branch `eip-8297-tests`),
by embedding the same allocation through `ethereum.binary_trie` and hashing
it: both produce

    0xe7bd46a85d2eb58005274fca6a7e721885e3d6220545cb9e2f8a057066f46a48

so this fixture pins the key derivation, the code chunking and the tree
hashing against the spec rather than against geth's own output.
