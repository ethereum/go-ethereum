Three signed transactions, used by the `t9n` (transaction validation) tests:

1. A legacy transaction with a gas limit of 5000000000, above both the EIP-7825
   cap (16777216, active with Osaka) and the EIP-8037 total cap (4294967295,
   active with Amsterdam).
2. A set code (EIP-7702) transaction with an empty authorization list.
3. A set code transaction with one valid authorization, as a control.

Transactions 1 and 2 are rejected by block execution, so the validator must
report an error for them under either fork. Transaction 3 is valid.

`exp.json` is the expected output on Amsterdam, `exp2.json` on Osaka.
