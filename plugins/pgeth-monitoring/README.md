# pgeth-monitoring plugin

Runs custom tracers while simulating all transactions, encode and feed everything to redis with topics making things easy to subscribe.

```
/head/tx/0xb0ba6c81c185bf7652f9339fdd86f35e47aea38a1215aa107f97d26ca5806c62/0x68c4D9E03D7D902053C428Ca2D74b612Db7F583A/0x5954aB967Bc958940b7EB73ee84797Dc8a2AFbb9/C@0x5954aB967Bc958940b7EB73ee84797Dc8a2AFbb9_20a325d0[S@0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D_6352211e,C@0x4d224452801ACEd8B2F0aebE155379bb5D594381_a9059cbb]
```

The topics follow the following format

```
/CHANNEL/tx/TX_HASH/FROM/TO/CALL_TRACES
```

Where the example above call traces can be interpreted as:

```
C@0x5954aB967Bc958940b7EB73ee84797Dc8a2AFbb9_20a325d0[S@0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D_6352211e,C@0x4d224452801ACEd8B2F0aebE155379bb5D594381_a9059cbb]

C@0x5954aB967Bc958940b7EB73ee84797Dc8a2AFbb9_20a325d0[
  S@0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D_6352211e,
  C@0x4d224452801ACEd8B2F0aebE155379bb5D594381_a9059cbb
]
```

- Initial call (`C`) made to `0x5954aB967Bc958940b7EB73ee84797Dc8a2AFbb9` on function selector `20a325d0`
  - Static call (`S`) made to `0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D` on function selector `6352211e`
  - Call (`C`) made to `0x4d224452801ACEd8B2F0aebE155379bb5D594381` on function selector `a9059cbb`


The different call modes are 

- `C`, a regular `call`
- `S`, a `staticcall`
- `D`, a `delegatecall`

The message payload will provide complete execution details with inputs, outputs, context and code address for every step
