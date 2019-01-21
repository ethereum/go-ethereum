## Whisper Overview

Whisper is a pure identity-based messaging system. Whisper provides a simple low-level API without being based upon or influenced by the low-level hardware attributes and characteristics. Peer-to-peer communication between the nodes of Whisper network uses the underlying [ÐΞVp2p Wire Protocol](https://github.com/ethereum/wiki/wiki/%C3%90%CE%9EVp2p-Wire-Protocol). Whisper was not designed to provide a connection-oriented system, nor for simply delivering data between a pair of particular network endpoints. However, this might be necessary in some very specific cases (e.g. delivering the expired messages in case they were missed), and Whisper protocol will accommodate for that. Whisper is designed for easy and efficient broadcasting, and also for low-level asynchronous communications. It is designed to be a building block in next generation of unstoppable ÐApps. It was designed to provide resilience and privacy at considerable expense. At its most secure mode of operation, Whisper can theoretically deliver 100% darkness. Whisper should also allow the users to configure the level of privacy (how much information it leaks concerning the ÐApp content and ultimately, user activities) as a trade-off for performance. 

Basically, all Whisper messages are supposed to be sent to every Whisper node. In order to prevent a DDoS attack, proof-of-work (PoW) algorithm is used. Messages will be processed (and forwarded further) only if their PoW exceeds a certain threshold, otherwise they will be dropped. 

### Encryption in version 5

All Whisper messages are encrypted and then sent via underlying ÐΞVp2p Protocol, which in turn uses its own encryption, on top of Whisper encryption. Every Whisper message must be encrypted either symmetrically or asymmetrically. Messages could be decrypted by anyone who possesses the corresponding key. 

In previous versions unencrypted messages were allowed, but since it was necessary to exchange the topic (more on that below), the nodes might as well use the same communication channel to exchange encryption key. 

Every node may possess multiple symmetric and asymmetric keys. Upon Envelope receipt, the node should try to decrypt it with each of the keys, depending on Envelope's encryption mode -- symmetric or asymmetric. In case of success, decrypted message is passed to the corresponding Ðapp. In any case, every Envelope should be forwarded to each of the node's peers.

Asymmetric encryption uses the standard Elliptic Curve Integrated Encryption Scheme with SECP-256k1 public key. Symmetric encryption uses AES GCM algorithm with random 96-bit nonce. If the same nonce will be used twice, then all the previous messages encrypted with the same key will be compromised. Therefore no more than 2^48 messages should be encrypted with the same symmetric key (for detailed explanation please see the Birthday Paradox). However, since Whisper uses proof-of-work, this number could possibly be reached only under very special circumstances (e.g. private network with extremely high performance and reduced PoW). Still, usage of one-time session keys is strongly encouraged for all Ðapps.

Although we assume these standard encryption algorithms to be reasonably secure, users are encouraged to use their own custom encryption on top of the default Whisper encryption.

#### Envelopes

Envelopes are the packets sent and received by Whisper nodes. Envelopes contain the encrypted payload and some metadata in plain format, because these data is essential for decryption. Envelopes are transmitted as RLP-encoded structures of the following format:

	[ Version, Expiry, TTL, Topic, AESNonce, Data, EnvNonce ]

Version: up to 4 bytes (currently one byte containing zero). Version indicates encryption method. If Version is higher than current, envelope could not be decrypted, and therefore only forwarded to the peers.

Expiry time: 4 bytes (UNIX time in seconds).

TTL: 4 bytes (time-to-live in seconds).

Topic: 4 bytes of arbitrary data.

AESNonce: 12 bytes of random data (only present in case of symmetric encryption).

Data: byte array of arbitrary size (contains encrypted message).

EnvNonce: 8 bytes of arbitrary data (used for PoW calculation).

Whisper nodes know nothing about content of envelopes which they can not decrypt. The nodes pass envelopes around regardless of their ability to decrypt the message, or their interest in it at all. This is an important component in Whisper's dark communications strategy.

#### Messages

Message is the content of Envelope's payload in plain format (unencrypted).

Plaintext (unencrypted) message is formed as a concatenation of a single byte for flags, additional metadata (as stipulated by the flags) and the actual payload. The message has the following structure:

    flags: 1 byte
    optional padding: byte array of arbitrary size
    payload: byte array of arbitrary size
    optional signature: 65 bytes

Those unable to decrypt the message data are also unable to access the signature. The signature, if provided, is the ECDSA signature of the Keccak-256 hash of the unencrypted data using the secret key of the originator identity. The signature is serialised as the concatenation of the `R`, `S` and `V` parameters of the SECP-256k1 ECDSA signature, in that order. `R` and `S` are both big-endian encoded, fixed-width 256-bit unsigned. `V` is an 8-bit big-endian encoded, non-normalised and should be either 27 or 28. 

The padding is introduced in order to align the message size, since message size alone might reveal important metainformation. The padding is supposed to contain random data (at least this is the default behaviour in version 5). However, it is possible to set arbitrary padding data, which might even be used for steganographic purposes. The API allows easy access to the padding data.

Default version 5 implementation ensures the size of the message to be multiple of 256. However, it is possible to set arbitrary padding through Whisper API. It might be useful for private Whisper networks, e.g. in order to ensure that all Whisper messages have the same (arbitrary) size. Incoming Whisper messages might have arbitrary padding size, and still be compatible with version 5.

The first several bytes of padding (up to four bytes) indicate the total size of padding. E.g. if padding is less than 256 bytes, then one byte is enough; if padding is less than 65536 bytes, then 2 bytes; and so on.

Flags byte uses only three bits in v.5. First two bits indicate, how many bytes indicate the padding size. The third byte indicates if signature is present. Other bits must be set to zero for backwards compatibility of future versions. 

#### Topics

It might not be feasible to try to decrypt ALL incoming envelopes, because decryption is quite expensive. In order to facilitate the filtering, Topics were introduced to the Whisper protocol. Topic gives a probabilistic hint about encryption key. Single Topic corresponds to a single key (symmetric or asymmetric). 

Upon receipt of a message, if the node detects a known Topic, it tries to decrypt the message with the corresponding key. In case of failure, the node assumes that Topic collision occurs, e.g. the message was encrypted with another key, and should be just forwarded further. Collisions are not only expected, they are necessary for plausible deniability.

Any Envelope could be encrypted only with one key, and therefore it contains only one Topic. 

Topic field contains 4 bytes of arbitrary data. It might be generated from the key (e.g. first 4 bytes of the key hash), but we strongly discourage it. In order to avoid any compromise on security, Topics should be completely unrelated to the keys. 

In order to use symmetric encryption, the nodes must exchange symmetric keys via some secure channel anyway. They might use the same channel in order to exchange the corresponding Topics as well. 

In case of asymmetric encryption, it might be more complicated since public keys are meant to be exchanged via the open channels. So, the Ðapp has a choice of either publishing its Topic along with the public key (thus compromising on privacy), or trying to decrypt all asymmetrically encrypted Envelopes (at considerable expense). Alternatively, PoW requirement for asymmetric Envelopes might be set much higher than for symmetric ones, in order to limit the number of futile attempts.

It is also possible to publish a partial Topic (first bytes), and then filter the incoming messages correspondingly. In this case the sender should set the first bytes as required, and rest should be randomly generated.

Examples:

	Partial Topic: 0x12 (first byte must be 0x12, the last three bytes - random)
	Partial Topic: 0x1234 (first two bytes must be {0x12, 0x34}, the last two bytes - random)
	Partial Topic: 0x123456 (first three bytes must be {0x12, 0x34, 0x56}, the last byte - random)

#### Filters

Any Ðapp can install multiple Filters utilising the Whisper API. Filters contain the secret key (symmetric or asymmetric), and some conditions, according to which the Filter should try to decrypt the incoming Envelopes. If Envelope does not satisfy those conditions, it should be ignored. Those are:
- array of possible Topics (or partial Topics)
- Sender address 
- Recipient address 
- PoW requirement
- AcceptP2P: boolean value, indicating whether the node accepts direct messages from trusted peers (reserved for some specific purposes, like Client/MailServer implementation)

All incoming messages, that have satisfied the Filter conditions AND have been successfully decrypted, will be saved by the corresponding Filter until the Ðapp requests them. Ðapps are expected to poll for incoming messages at regular time intervals. All installed Filters are independent of each other, and their conditions might overlap. If a message satisfies the conditions of multiple Filters, it will be stored in each of the Filters.

In future versions subscription will be used instead of polling.

In case of partial Topic, the message will match the Filter if first X bytes of the message Topic are equal to the corresponding bytes of partial Topic in Filter (where X can be 1, 2 or 3). The last bytes of the message Topic are ignored in this case.

#### Proof of Work

The purpose of PoW is spam prevention, and also reducing the burden on the network. The cost of computing PoW can be regarded as the price you pay for allocated resources if you want the network to store your message for a specific time (TTL). In terms of resources, it does not matter if the network stores X equal messages for Y seconds, or Y messages for X seconds. Or N messages of Z bytes each versus Z messages of N bytes. So, required PoW should be proportional to both message size and TTL.

After creating the Envelope, its Nonce should be repeatedly incremented, and then its hash should be calculated. This procedure could be run for a specific predefined time, looking for the lowest hash. Alternatively, the node might run the loop until certain predefined PoW is achieved.

In version 5, PoW is defined as average number of iterations, required to find the current BestBit (the number of leading zero bits in the hash), divided by message size and TTL:

<code>PoW = (2^BestBit) / (size * TTL)</code>

Thus, we can use PoW as a single aggregated parameter for the message rating. In the future versions every node will be able to set its own PoW requirement dynamically and communicate this change to the other nodes via the Whisper protocol. Now it is only possible to set PoW requirement at the Ðapp startup.

### Packet Codes (ÐΞVp2p level)

As a sub-protocol of [ÐΞVp2p](https://github.com/ethereum/wiki/wiki/%C3%90%CE%9EVp2p-Wire-Protocol), Whisper sends and receives its messages within ÐΞVp2p packets. 
Whisper v5 supports the following packet codes:

<code>Status (0x0)</code>

<code>Messages (0x1)</code>

<code>P2PMessage (0x2)</code>

<code>P2PRequest (0x3)</code>

Also, the following codes might be supported in the future:

<code>PoWRequirement (0x4)</code>

<code>BloomFilterExchange (0x5)</code>

### Basic Operation

Nodes are expected to receive and send envelopes continuously. They should maintain a map of envelopes, indexed by expiry time, and prune accordingly. They should also efficiently deliver messages to the front-end API through maintaining mappings between Ðapps, their filters and envelopes.

When a node's envelope memory becomes exhausted, a node may drop envelopes it considers unimportant or unlikely to please its peers. Nodes should rate peers higher if they pass them envelopes with higher PoW. Nodes should blacklist peers if they pass invalid envelopes, i.e., expired envelopes or envelopes with an implied insertion time in the future.

Nodes should always treat messages that its ÐApps have created no different than incoming messages.

#### Creating and Sending Messages

To send a message, the node should place the envelope its envelope pool. Then this envelope will be forwarded to the peers in due course along with the other envelopes. Composing an envelope from a basic payload, is done in a few steps:

- Compose the Envelope data by concatenating the relevant flag byte, padding, payload (randomly generated or provided by user), and an optional signature.
- Encrypt the data symmetrically or asymmetrically.
- Add a Topic.
- Set the TTL attribute.
- Set the expiry as the present Unix time plus TTL.
- Set the nonce which provides the best PoW.

### Mail Server

Suppose, a Ðapp waits for messages with certain Topic and suffers an unexpected network failure for certain period of time. As a result, a number of important messages will be lost. Since those messages are expired, there is no way to resend them via the normal Whisper channels, because they will be rejected and the peer punished.

One possible way to solve this problem is to run a Mail Server, which would store all the messages, and resend them at the request of the known nodes. Even though the server might repack the old messages and provide sufficient PoW, it's not feasible to resend all of them at whim, because it would tantamount to DDoS attack on the entire network. Instead, the Mail Server should engage in peer-to-peer communication with the node, and resend the expired messages directly. The recipient will consume the messages and will not forward them any further. 

In order to facilitate this task, protocol-level support is provided in version 5. New message types are introduced to Whisper v.5: mailRequestCode and p2pCode.

- mailRequestCode is used by the node to request historic (expired) messages from the Mail Server. The payload of this message should be understood by the Server. The Whisper protocol is entirely agnostic about it. It might contain a time frame, the node's authorization details, filtering information, payment details, etc.

- p2pCode is a peer-to-peer message, that is not supposed to be forwarded to other peers. It will also bypass the protocol-level checks for expiry and PoW threshold.

There is MailServer interface defined in the codebase for easy Mail Server implementation. One only needs to implement two functions:

type MailServer interface {
	Archive(env *Envelope)
	DeliverMail(whisperPeer *Peer, data []byte)
}

Archive should just save all incoming messages.
DeliverMail should be able to process the request and send the historic messages to the corresponding peer (with p2pCode), according to additional information in the data parameter. This function will be invoked upon receipt of protocol-level mailRequestCode.
