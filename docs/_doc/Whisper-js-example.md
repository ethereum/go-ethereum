---
title: Whisper JavaScript example
---

[This link](https://github.com/gballet/whisper-chat-example) contains a full-fledged example of how to use Whisper in a small chat application.

Let's now have a look at the `sendMessage` function:

```javascript
sendMessage() {
    // Start by declaring the message, we picked a JSON format with
    // `text` as the content and `name` as the name of the user who
    // is sending the message.
    let msg = {
        text: this.text,
        name: this.name
    };

    // (code elided for clarity)

    // Create the data object that will be sent to the RPC endpoint.
    let postData = {
        ttl: 7,
        topic: '0x07678231',
        powTarget: 2.01,
        powTime: 100,
        payload: encodeToHex(JSON.stringify(msg)),
    };

    // Set the appropriate key id.
    if (this.asym) {
        postData.pubKey = this.recipientPubKey;
        postData.sig = this.asymKeyId;
    } else
        postData.symKeyID = this.symKeyId;

    // Perform the RPC call that will tell the node to forward
    // that message to all its neighboring nodes.
    this.shh.post(postData);

    // (code elided for clarity)
}
```