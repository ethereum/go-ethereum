---
title: Whisper RPC API 6.0
sort_key: C
---

This is the proposed API for whisper v6.

### Specs

- [shh_version](#shh_version)
- [shh_info](#shh_info)
- [shh_setMaxMessageSize](#shh_setmaxmessagesize)
- [shh_setMinPoW](#shh_setminpow)
- [shh_markTrustedPeer](#shh_marktrustedpeer)
- [shh_newKeyPair](#shh_newkeypair)
- [shh_addPrivateKey](#shh_addprivatekey)
- [shh_deleteKeyPair](#shh_deletekeypair)
- [shh_hasKeyPair](#shh_haskeypair)
- [shh_getPublicKey](#shh_getpublickey)
- [shh_getPrivateKey](#shh_getprivatekey)
- [shh_newSymKey](#shh_newsymkey)
- [shh_addSymKey](#shh_addsymkey)
- [shh_generateSymKeyFromPassword](#shh_generatesymkeyfrompassword)
- [shh_hasSymKey](#shh_hassymkey)
- [shh_getSymKey](#shh_getsymkey)
- [shh_deleteSymKey](#shh_deletesymkey)
- [shh_subscribe](#shh_subscribe)
- [shh_unsubscribe](#shh_unsubscribe)
- [shh_newMessageFilter](#shh_newmessagefilter)
- [shh_deleteMessageFilter](#shh_deletemessagefilter)
- [shh_getFilterMessages](#shh_getfiltermessages)
- [shh_post](#shh_post)


***

#### shh_version

Returns the current semver version number.

##### Parameters

none

##### Returns

`String` - The version number.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_version","params":[],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "6.0"
}
```

***

#### shh_info

Returns diagnostic information about the whisper node.

##### Parameters

none

##### Returns

`Object` - diagnostic information with the following properties:
  - `minPow` - `Number`: current minimum PoW requirement.
  - `maxMessageSize` - `Float`: current messgae size limit in bytes.
  - `memory` - `Number`: Memory size of the floating messages in bytes.
  - `messages` - `Number`: Number of floating messages.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_info","params":[],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": {
    "minPow": 12.5,
    "maxMessageSize": 20000,
    "memory": 10000,
    "messages": 20,
  }
}
```

***

#### shh_setMaxMessageSize

Sets the maximal message size allowed by this node.
Incoming and outgoing messages with a larger size will be rejected.
Whisper message size can never exceed the limit imposed by the underlying P2P protocol (10 Mb).

##### Parameters

1. `Number`: Message size in bytes.

##### Returns

`Boolean`: (`true`) on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_setMaxMessageSize","params":[234567],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": true
}
```

***

#### shh_setMinPoW

Sets the minimal PoW required by this node.

This experimental function was introduced for the future dynamic adjustment of PoW requirement. If the node is overwhelmed with messages, it should raise the PoW requirement and notify the peers. The new value should be set relative to the old value (e.g. double). The old value could be obtained via shh_info call.

**Note** This function is currently experimental.

##### Parameters

1. `Number`: The new PoW requirement.

##### Returns

`Boolean`: `true` on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_setMinPoW","params":[12.3],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": true
}
```

***

#### shh_markTrustedPeer

Marks specific peer trusted, which will allow it to send historic (expired) messages.

**Note** This function is not adding new nodes, the node needs to exists as a peer.

##### Parameters

1. `String`: Enode of the trusted peer.

##### Returns

`Boolean` (`true`) on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_markTrustedPeer","params":["enode://d25474361659861e9e651bc728a17e807a3359ca0d344afd544ed0f11a31faecaf4d74b55db53c6670fd624f08d5c79adfc8da5dd4a11b9213db49a3b750845e@52.178.209.125:30379"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": true
}
```

***


#### shh_newKeyPair

Generates a new public and private key pair for message decryption and encryption.

##### Parameters

none

##### Returns

`String`: Key ID on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_newKeyPair","id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "5e57b9ffc2387e18636e0a3d0c56b023264c16e78a2adcba1303cefc685e610f"
}
```

***


#### shh_addPrivateKey

Stores the key pair, and returns its ID.

##### Parameters

1. `String`: private key as HEX bytes.

##### Returns

`String`: Key ID on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_addPrivateKey","params":["0x8bda3abeb454847b515fa9b404cede50b1cc63cfdeddd4999d074284b4c21e15"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "3e22b9ffc2387e18636e0a3d0c56b023264c16e78a2adcba1303cefc685e610f"
}
```

***

#### shh_deleteKeyPair

Deletes the specifies key if it exists.

##### Parameters

1. `String`: ID of key pair.

##### Returns

`Boolean`: `true` on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_deleteKeyPair","params":["5e57b9ffc2387e18636e0a3d0c56b023264c16e78a2adcba1303cefc685e610f"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": true
}
```

***


#### shh_hasKeyPair

Checks if the whisper node has a private key of a key pair matching the given ID.

##### Parameters

1. `String`: ID of key pair.

##### Returns

`Boolean`: (`true` or `false`) and error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_hasKeyPair","params":["5e57b9ffc2387e18636e0a3d0c56b023264c16e78a2adcba1303cefc685e610f"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": false
}
```

***

#### shh_getPublicKey

Returns the public key for identity ID.

##### Parameters

1. `String`: ID of key pair.

##### Returns

`String`: Public key on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_getPublicKey","params":["86e658cbc6da63120b79b5eec0c67d5dcfb6865a8f983eff08932477282b77bb"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x04d1574d4eab8f3dde4d2dc7ed2c4d699d77cbbdd09167b8fffa099652ce4df00c4c6e0263eafe05007a46fdf0c8d32b11aeabcd3abbc7b2bc2bb967368a68e9c6"
}
```

***

#### shh_getPrivateKey

Returns the private key for identity ID.

##### Parameters

1. `String`: ID of the key pair.

##### Returns

`String`: Private key on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_getPrivateKey","params":["0xc862bf3cf4565d46abcbadaf4712a8940bfea729a91b9b0e338eab5166341ab5"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x234234e22b9ffc2387e18636e0534534a3d0c56b0243567432453264c16e78a2adc"
}
```

***

#### shh_newSymKey

Generates a random symmetric key and stores it under an ID, which is then returned.
Can be used encrypting and decrypting messages where the key is known to both parties.

##### Parameters

none

##### Returns

`String`: Key ID on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_newSymKey", "params": [], "id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "cec94d139ff51d7df1d228812b90c23ec1f909afa0840ed80f1e04030bb681e4"
}
```

***

#### shh_addSymKey

Stores the key, and returns its ID.

##### Parameters

1. `String`: The raw key for symmetric encryption as HEX bytes.

##### Returns

`String`: Key ID on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_addSymKey","params":["0xf6dcf21ed6a17bd78d8c4c63195ab997b3b65ea683705501eae82d32667adc92"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "5e57b9ffc2387e18636e0a3d0c56b023264c16e78a2adcba1303cefc685e610f"
}
```

***

#### shh_generateSymKeyFromPassword

Generates the key from password, stores it, and returns its ID.

##### Parameters

1. `String`: password.

##### Returns

`String`: Key ID on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_generateSymKeyFromPassword","params":["test"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "2e57b9ffc2387e18636e0a3d0c56b023264c16e78a2adcba1303cefc685e610f"
}
```

***

#### shh_hasSymKey

Returns true if there is a key associated with the name string. Otherwise, returns false.

##### Parameters

1. `String`: key ID.

##### Returns

`Boolean` (`true` or `false`) on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_hasSymKey","params":["f6dcf21ed6a17bd78d8c4c63195ab997b3b65ea683705501eae82d32667adc92"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": true
}
```


***

#### shh_getSymKey

Returns the symmetric key associated with the given ID.

##### Parameters

1. `String`: key ID.

##### Returns

`String`: Raw key on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_getSymKey","params":["f6dcf21ed6a17bd78d8c4c63195ab997b3b65ea683705501eae82d32667adc92"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "0xa82a520aff70f7a989098376e48ec128f25f767085e84d7fb995a9815eebff0a"
}
```


***

#### shh_deleteSymKey

Deletes the key associated with the name string if it exists.

##### Parameters

1. `String`: key ID.

##### Returns

`Boolean` (`true` or `false`) on success and an error on failure.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_deleteSymKey","params":["5e57b9ffc2387e18636e0a3d0c56b023264c16e78a2adcba1303cefc685e610f"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": true
}
```

***

#### shh_subscribe

Creates and registers a new subscription to receive notifications for inbound whisper messages. Returns the ID of the newly created subscription.

##### Parameters

1. `id` - `String`: identifier of function call. In case of Whisper must contain the value "messages".

2. `Object`. Options object with the following properties:
  - `symKeyID` - `String`: ID of symmetric key for message decryption.
  - `privateKeyID` - `String`: ID of private (asymmetric) key for message decryption.
  - `sig` - `String`  (optional): Public key of the signature.
  - `minPow` - `Number`  (optional): Minimal PoW requirement for incoming messages.
  - `topics` - `Array`  (optional when asym key): Array of possible topics (or partial topics).
  - `allowP2P` - `Boolean`  (optional): Indicates if this filter allows processing of direct peer-to-peer messages (which are not to be forwarded any further, because they might be expired). This might be the case in some very rare cases, e.g. if you intend to communicate to MailServers, etc.

Either `symKeyID` or `privateKeyID` must be present. Can not be both.

##### Returns

`String` - The subscription ID on success, the error on failure.


##### Notification Return

`Object`: The whisper message matching the subscription options, with the following parameters:
  - `sig` - `String`: Public key who signed this message.
  - `recipientPublicKey` - `String`: The recipients public key.
  - `ttl` - `Number`: Time-to-live in seconds.
  - `timestamp` - `Number`: Unix timestamp of the message genertion.
  - `topic` - `String` 4 Bytes: Message topic.
  - `payload` - `String`: Decrypted payload.
  - `padding` - `String`: Optional padding (byte array of arbitrary length).
  - `pow` - `Number`: Proof of work value.
  - `hash` - `String`: Hash of the enveloved message.


##### Example
```
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_subscribe","params":["messages", {
  topics: ['0x5a4ea131', '0x11223344'],
  symKeyID: 'b874f3bbaf031214a567485b703a025cec27d26b2c4457d6b139e56ad8734cea',
  sig: '0x048229fb947363cf13bb9f9532e124f08840cd6287ecae6b537cda2947ec2b23dbdc3a07bdf7cd2bfb288c25c4d0d0461d91c719da736a22b7bebbcf912298d1e6',
  pow: 12.3(?)
  }],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": "02c1f5c953804acee3b68eda6c0afe3f1b4e0bec73c7445e10d45da333616412"
}


// Notification Result
{
  "jsonrpc": "2.0",
  "method": "shh_subscription",
  "params": {
    subscription: "02c1f5c953804acee3b68eda6c0afe3f1b4e0bec73c7445e10d45da333616412",
    result: {
      sig: '0x0498ac1951b9078a0549c93c3f6088ec7c790032b17580dc3c0c9e900899a48d89eaa27471e3071d2de6a1f48716ecad8b88ee022f4321a7c29b6ffcbee65624ff',
      recipientPublicKey: null,
      ttl: 10,
      timestamp: 1498577270,
      topic: '0xffaadd11',
      payload: '0xffffffdddddd1122',
      padding: '0x35d017b66b124dd4c67623ed0e3f23ba68e3428aa500f77aceb0dbf4b63f69ccfc7ae11e39671d7c94f1ed170193aa3e327158dffdd7abb888b3a3cc48f718773dc0a9dcf1a3680d00fe17ecd4e8d5db31eb9a3c8e6e329d181ecb6ab29eb7a2d9889b49201d9923e6fd99f03807b730780a58924870f541a8a97c87533b1362646e5f573dc48382ef1e70fa19788613c9d2334df3b613f6e024cd7aadc67f681fda6b7a84efdea40cb907371cd3735f9326d02854',
      pow: 0.6714754098360656,
      hash: '0x17f8066f0540210cf67ef400a8a55bcb32a494a47f91a0d26611c5c1d66f8c57'
    }
  }
}
```

***

#### shh_unsubscribe

Cancels and removes an existing subscription.

##### Parameters

1. `String`: subscription ID.

##### Returns

`Boolean`: `true` or `false`.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_unsubscribe","params":["02c1f5c953804acee3b68eda6c0afe3f1b4e0bec73c7445e10d45da333616412"],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": true
}
```

***

#### shh_newMessageFilter

Create a new filter within the node. This filter can be used to poll for new messages that match the set of criteria.

##### Parameters

1. `criteria` - `Object`: see [shh_subscribe](#shh_subscribe)
 
##### Returns

`String`: filter identifier

##### Example
```
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_newMessageFilter","params":[{"symKeyID": "3742c75e4232325d54143707e4b73d17c2f86a5e4abe3359021be5653f5b2c81"}],"id":1}' localhost:8545

// Result
{"jsonrpc":"2.0","id":1,"result":"2b47fbafb3cce24570812a82e6e93cd9e2551bbc4823f6548ff0d82d2206b326"}

```


***

#### shh_deleteMessageFilter

Uninstall a message filter in the node

##### Parameters

1. `id` - `String`: filter identifier as returned when the filter was created
 
##### Returns

`Boolean`: `true` on success, error on failure.

##### Example
```
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_deleteMessageFilter","params":[{"symKeyID": "3742c75e4232325d54143707e4b73d17c2f86a5e4abe3359021be5653f5b2c81"}],"id":1}' localhost:8545

// Result
{"jsonrpc":"2.0","id":1,"result": true}

```


***

#### shh_getFilterMessages

Retrieve messages that match the filter criteria and are received between the last time this
function was called and now.

##### Parameters

1. `id` - `String`: ID of filter that was created with `shh_newMessageFilter`
 

##### Returns

`Array of messages`: `true` on success and an error on failure.

`Object`: whisper message:
  - `sig` - `String`: Public key who signed this message.
  - `ttl` - `Number`: Time-to-live in seconds.
  - `timestamp` - `Number`: Unix timestamp of the message generation.
  - `topic` - `String` 4 Bytes: Message topic.
  - `payload` - `String`: Decrypted payload.
  - `padding` - `String`: Optional padding (byte array of arbitrary length).
  - `pow` - `Number`: Proof of work value.
  - `hash` - `String`: Hash of the enveloped message.
  - `recipientPublicKey` - `String`: recipient public key

##### Example
```
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_getFilterMessages","params":["2b47fbafb3cce24570812a82e6e93cd9e2551bbc4823f6548ff0d82d2206b326"],"id":1}'

// Result
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": [
    {
      "hash": "0xe05c4be74d667bd4c57dba2a8dbfb097d6fc2719d5c0d699d2f84a2442a4d8c2",
      "padding": "0x6e3e82571c7aa91f2a9e82e20344ede0d1112205555843d9dafffeb1536741a1fca758ff30cc01320bb0775aa5b82b9c9f48deb10bff8b5c61354bf8f95f2ab7289c7894c52e285b1d750ea2ffa78edd374bc7386a901d36a59022d73208c852dedaca8709087693ef6831b861139f42a89263af5931cb2b9253216dc42edc1393afd03940f91c561d20080f7a258aa52d30dcf4b1b921b7c910ad429f73ed9308cb6218537f0444d915e993ba8c9bb00af311aab3574bf1722b5640632bf5bb6b12406e1b89d0c98628117b1d8f55ea6b974e251b34969d4c49dfb6036d40e0d90414c65a8b036ae985396d6a4bf28332676e85dc0a0d352a58680200cc",
      "payload": "0xabcd",
      "pow": 0.5371803278688525,
      "recipientPublicKey": null,
      "sig": null,
      "timestamp": 1496991875,
      "topic": "0x01020304",
      "ttl": 50
    },
    {
      "hash": "0x4158eb81ad8e30cfcee67f20b1372983d388f1243a96e39f94fd2797b1e9c78e",
      "padding": "0xc15f786f34e5cef0fef6ce7c1185d799ecdb5ebca72b3310648c5588db2e99a0d73301c7a8d90115a91213f0bc9c72295fbaf584bf14dc97800550ea53577c9fb57c0249caeb081733b4e605cdb1a6011cee8b6d8fddb972c2b90157e23ba3baae6c68d4f0b5822242bb2c4cd821b9568d3033f10ec1114f641668fc1083bf79ebb9f5c15457b538249a97b22a4bcc4f02f06dec7318c16758f7c008001c2e14eba67d26218ec7502ad6ba81b2402159d7c29b068b8937892e3d4f0d4ad1fb9be5e66fb61d3d21a1c3163bce74c0a9d16891e2573146aa92ecd7b91ea96a6987ece052edc5ffb620a8987a83ac5b8b6140d8df6e92e64251bf3a2cec0cca",
      "payload": "0xdeadbeaf",
      "pow": 0.5371803278688525,
      "recipientPublicKey": null,
      "sig": null,
      "timestamp": 1496991876,
      "topic": "0x01020304",
      "ttl": 50
    }
  ]
}

```







***

#### shh_post

Creates a whisper message and injects it into the network for distribution.

##### Parameters

1. `Object`. Post options object with the following properties:
  - `symKeyID` - `String`: ID of symmetric key for message encryption.
  - `pubKey` - `String`: public key for message encryption.
  - `sig` - `String` (optional): ID of the signing key.
  - `ttl` - `Number`: Time-to-live in seconds.
  - `topic` - `String` 4 Bytes (mandatory when key is symmetric): Message topic.
  - `payload` - `String`: Payload to be encrypted.
  - `padding` - `String` (optional): Optional padding (byte array of arbitrary length).
  - `powTime` - `Number`: Maximal time in seconds to be spent on proof of work.
  - `powTarget` - `Number`: Minimal PoW target required for this message.
  - `targetPeer` - `String` (optional): Optional peer ID (for peer-to-peer message only).

Either `symKeyID` or `pubKey` must be present. Can not be both.

##### Returns

`Boolean`: `true` on success and an error on failure.

##### Example
```
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_post","params":[{
  pubKey: 'b874f3bbaf031214a567485b703a025cec27d26b2c4457d6b139e56ad8734cea',
  ttl: 7,
  topic: '0x07678231',
  powTarget: 2.01,
  powTime: 2,
  payload: '0x68656c6c6f'
  }],"id":1}'

// Result
{
  "id":1,
  "jsonrpc": "2.0",
  "result": true
}
```
