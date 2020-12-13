---
title: Whisper JavaScript example
sort_key: B
---

[This link](https://github.com/gballet/whisper-chat-example) contains a full-fledged example of how to use Whisper in a small chat application.

The app is a simple Vue single page application that works in several steps. In the first step, the user configures the RPC service that they wish to connect to. Typically, this would be a `geth` client with the `--shh` option enabled.

Then one can install the application locally by typing:

```
$ git clone https://github.com/gballet/whisper-chat-example
$ cd whisper-chat-example
$ npm install
```

The application is then started by typing:

```
$ npm run dev
```

The application will then be available at `http://localhost:8080` (Note the http here, which is because it's a **demo application** and should not be used in production)

## User workflow

The app starts by asking the user for their user name and whether they want a symmetric or asymmetric connection. If it's asymmetric, the geth server will propose a public key. If it's symmetric, a key has to be provided.

Then the user presses "Start" and the conversation begins.

## The app's state

```javascript
                let data = {
                        msgs: [],
                        text: "",
                        symKeyId: null,
                        name: "",
                        asymKeyId: null,
                        sympw: "",
                        asym: true,
                        configured: false,
                        topic: defaultTopic,
                        recipientPubKey: defaultRecipientPubKey,
                        asymPubKey: ""
                };
```

This is how the current, transient, state of the application is represented:

  * `msgs` is the list of messages in the current conversation
  * `text` contains the text that the current user is typing
  * `name` is the name of the current user, which is used to identify them in conversations
  * `asymKeyId` and `symKeyId` represent handles to the corresponding keys in `geth`'s memory
  * `recipientPubKey` is a hex string representing the public key that an asymmetric message is sent to
  * `topic` is a hex string representing the message's topic
  * `asymPubKey` is a hex string representing the user's own public key
  * `configured` is a flag that is set to true when the user has choosen either a public key or a symmetric key, and a user name
  * `sympw` contains the symmetric password

## The `sendMessage` callback

The `sendMessage` callback is called every time the user clicks on "Send" or presses the return key. It is responsible for creating the RPC request that instructs the `geth` node to encrypt and send the message.

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
    // ...

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
    // ...
}
```

The `msg` object is created. The format chosen for the object is specific to this demo application. It just contains a text and the name of the sender. This is obviously not secure enough for a real-world application.

That object is converted to a string and then encoded as a hexadecimal string, in the `payload` member of the request's `POST` data object. Other fields include the `topic` of the message, how much work the sending server should do and other parameters.

Next, depending whether the "asymmetric" checkbox has been ticked, the value of `this.asym` will be true or false. Based on this, the system will update the request object with the relevant information.

Finally, the request is being sent with `this.shh.post(postData)`, which calls Web3's `shh.post` function to send the message.
