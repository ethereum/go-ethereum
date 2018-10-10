# Specs
`encode(domainSeparator : 𝔹²⁵⁶, message : 𝕊) = "\x19\x01" ‖ domainSeparator ‖ hashStruct(message)`  
- data adheres to 𝕊, a structure defined in the rigorous eip-712
- `\x01` is needed to comply with EIP-191
- `domainSeparator` and `hashStruct` are defined below

## A) domainSeparator
`domainSeparator = hashStruct(eip712Domain)`
<br/>
<br/>
Struct named `EIP712Domain` with one or more of the below fields:

- `string name`
- `string version`
- `uint256 chainId`, as per EIP-155
- `address verifyingContract`
- `bytes32 salt`

## B) hashStruct
`hashStruct(s : 𝕊) = keccak256(typeHash ‖ encodeData(s))`
<br/>
`typeHash = keccak256(encodeType(typeOf(s)))`

### i) encodeType
- `name ‖ "(" ‖ member₁ ‖ "," ‖ member₂ ‖ "," ‖ … ‖ memberₙ ")"`
- each member is written as `type ‖ " " ‖ name`
- encodings cascade down and are sorted by name

### ii) encodeData
- `enc(value₁) ‖ enc(value₂) ‖ … ‖ enc(valueₙ)`
- each encoded member is 32-byte long

    #### a) atomic

    - `boolean`     => `uint256`
    - `address`     => `uint160`
    - `uint`        => sign-extended `uint256` in big endian order
    - `bytes1:31`   => `bytes32` 

    #### b) dynamic

    - `bytes`       => `keccak256(bytes)`
    - `string`      => `keccak256(string)`

    #### c) referenced

    - `array`       => `keccak256(encodeData(array))`
    - `struct`      => `rec(keccak256(hashStruct(struct)))`

## C) Example
### Query
```json
{
  "jsonrpc": "2.0",
  "method": "account_signStructuredData",
  "params": [
    "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826",
    {
      "types": {
        "EIP712Domain": [
          {
            "name": "name",
            "type": "string"
          },
          {
            "name": "version",
            "type": "string"
          },
          {
            "name": "chainId",
            "type": "uint256"
          },
          {
            "name": "verifyingContract",
            "type": "address"
          }
        ],
        "Person": [
          {
            "name": "name",
            "type": "string"
          },
          {
            "name": "wallet",
            "type": "address"
          }
        ],
        "Mail": [
          {
            "name": "from",
            "type": "Person"
          },
          {
            "name": "to",
            "type": "Person"
          },
          {
            "name": "contents",
            "type": "string"
          }
        ]
      },
      "primaryType": "Mail",
      "domain": {
        "name": "Ether Mail",
        "version": "1",
        "chainId": 1,
        "verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
      },
      "message": {
        "from": {
          "name": "Cow",
          "wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
        },
        "to": {
          "name": "Bob",
          "wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
        },
        "contents": "Hello, Bob!"
      }
    }
  ],
  "id": 1
}
```

### Response
```json
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x4355c47d63924e8a72e509b65029052eb6c299d53a04e167c5775fd466751c9d07299936d304c153f6443dfa05f40ff007d72911b6f72307f996231605b915621c"
}
```