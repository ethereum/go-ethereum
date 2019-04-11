---
title: Diagnostic tool wnode
---
# Wnode

Wnode (whisper node) is a command-line diagnostic tool. It does not have a nice user interface, because its main purpose is diagnostic, and it's meant to be very light-weight rather than beautiful. Wnode might be used for different purposes, including:

- running a standalone bootstrap node, in order to set up a private whisper network
- connecting to particular node for debugging or other purposes
- engaging in command-line chat with another peer in order to test its functionality
- sending and receiving text messages
- sending and receiving files
- running fully functional mail server
- testing functionality of mail client

## Usage

```
> wnode [flags/arguments]
```
	
## Flags & Switches

In case an argument is missing, `wnode` will either use the default value or prompt the user at startup. For security reasons, it is not possible to provide passwords in command-line arguments. In `test` mode, a hardcoded password ("test") is used.

`-asym`: use asymmetric encryption in the chat

`-boot`: A string representing the bootstrap node to connect to

`-dbpath`: The path to the server's DB directory

`-echo`: prints some arguments for diagnostics

`-fileexchange`: file exchange mode (send and receive files instead of text messages)

`-filereader`: load and decrypt messages saved as files, display as plain text

`-forwarder`: forwarder mode (only forward; neither send nor decrypt messages)

`-generatekey`: generate and show the private key, and exit

`-idfile`: file name with node id (private key)

`-ip`: IP address and port of this node (e.g. 127.0.0.1:30303)

`-mailclient`: request expired messages from the mail server

`-mailserver`: mail server mode (delivers expired messages on demand)

`maxsize`: max size of message (default 1048576)

`-mspow`: PoW requirement for Mail Server request (default 0.2)

`-pow`: PoW for normal messages in float format (e.g. 2.7) (default 0.2)

`-pub`: public key for asymmetric encryption

`-savedir`: directory where all incoming messages will be saved as files

`-standalone`: don't actively connect to any peers, wait for incoming connections instead

`-test`: use of predefined parameters for diagnostics, including passwords

`-topic`: topic in hexadecimal format (e.g. 70a4beef)

`-ttl`: time-to-live for messages in seconds (default 30)

`-verbosity`: log verbosity level (default 1)

`-work`: work time in seconds (default 5)



-pub:
Public key of your peer (for asymmetric encryption). For example:
-pub=0x07af49cbe6353b8732a8b9eb20dd1472f3d4512cd1a11382ee2817cc6de9453bc07c32c730b93bc83877b11e4f47d718751297f4edcbf35015df2b34ff5fc6a75d

-idfile:
File name containing node ID (private key) in hexadecimal string format.
For example: -idfile=/home/vlad/tmp/config
Example of the file content: b3651aff593ef395ee7c16f3ca681830f7d8d0b2729cf472b14f2c4ebe833aa0

-boot:
The bootstrap node you want to connect to. For example:
-boot=enode://d25474361659861e9e651bc728a17e807a3359ca0d344afd544ed0f11a31faecaf4d74b55db53c6670fd624f08d5c79adfc8da5dd4a11b9213db49a3b750845e@52.178.209.125:30379

-topic:
Message topic in hexadecimal format. For example: -topic=70a4beef.

-dbpath:
Path to directory where Mail Server will store the incoming messages. 
For example: -dbpath=tmp/myfiles/archive

-savedir:
Directory where successfully decrypted incoming messages will be saved as files in plain format. 
Message hashes will be used as file names in order to avoid collisions.
By default, only big messages are stored there. 
In 'fileexchange' mode all messages are stored there.

## Scenarios & Examples

For simplicity, in these examples we assume that we only use `wnode` to communicate with another wnode.


### Start a bootstrap node for test network

```
> wnode -standalone -forwarder -ip=127.0.0.1:30379
my public key: 0x040ef7acd60781c336c52056b3782f7eae45be2063e591ac6b78472dc27ba770010bde445ffd2f3623ad656f3859e00d11ef518df4916c4d4e258c60b15f34c682	enode://15454fc65bbf0031155f4eee83fa732f1454c314e9f78ade9cba4d4a098d29edbf5431764ee65b200169025c3f900cacc3348a000dda7a8a0d9643d0b7618712@127.0.0.1:30379
Bootstrap Whisper node started
```

### Connecting to a bootstrap node

After the bootstrap node has started, another local node can connect to it, using the resulting enode:

```
> wnode -test -boot=enode://15454fc65bbf0031155f4eee83fa732f1454c314e9f78ade9cba4d4a098d29edbf5431764ee65b200169025c3f900cacc3348a000dda7a8a0d9643d0b7618712@127.0.0.1:30379
............................
Whisper node started
Connected to peer.
............................
```
	
Upon restarting the bootstrap node, its enode will be different, because the ID is randomly generated. For persistence accross restarts, it is possible to specify an ID stored in a file using the 'idfile' argument. 

Generating ID:

```
> wnode -generatekey > pk1.txt
```

`pk1.txt` now contains the key used to generate the ID.
	
Starting the bootstrap node with a persistent ID:

```
> wnode -forwarder -standalone -ip=127.0.0.1:30379 -idfile=pk1.txt
my public key: 0x04be81a00a90f5c21ead8887eaa254b3f7a37e06f8f2d776dcc46954a228bc50c6fb6dfd155f7e44e6fef9b62fdf6dad041759b864d2cbe4089b6f5c16a817ff46	enode://7d13360f5b1ddcf6947f244639113597a863abba0589d2fa5fffb2816ead0acea6211d5778a8be648e45e81ed881f4c1f5c9bbbf0e79065dfb54bcd97de3beab@127.0.0.1:30379
	Filter is configured for the topic: 5a4ea131 
	Bootstrap Whisper node started
```

Now you can always use the same command to connect to your bootstrap node:

```
> wnode -test -boot=enode://7d13360f5b1ddcf6947f244639113597a863abba0589d2fa5fffb2816ead0acea6211d5778a8be648e45e81ed881f4c1f5c9bbbf0e79065dfb54bcd97de3beab@127.0.0.1:30379
```
	
Be aware that the ID is stored unencrypted. This feature should only be used for test purposes.

In order to set up a bootstrap node on a server with a dedicated IP address, its IP and port need to be specified explicitly:

```
> wnode -forwarder -standalone -ip=52.178.211.103:30379
```

### Using a mail server

The mailserver is only provided as an example for people interested in building their own solution. It is not supported.

```
> wnode -forwarder -standalone -ip=127.0.0.1:30381 -idfile=config.txt -mailserver -dbpath=tmp/db
```


### Chat with symmetric encryption

For two nodes to communicate using symmetric encryption, one of them must assume the role of a bootstrap node, and the second one that of the client. The bootstrap node is started with the `standalone` flag, and the client must connect to it. It is easy to do on the same machine or on a dedicated server. But what if two peers are behind distinct NAT? In that case, you need a third bootstrap node on a dedicated server, which both peers can connect to. At the time of writing we have out test node with the following enode:
`enode://d25474361659861e9e651bc728a17e807a3359ca0d344afd544ed0f11a31faecaf4d74b55db53c6670fd624f08d5c79adfc8da5dd4a11b9213db49a3b750845e@52.178.209.125:30379`,
to which both peers can connect with the following command:

```
> wnode -boot=enode://d25474361659861e9e651bc728a17e807a3359ca0d344afd544ed0f11a31faecaf4d74b55db53c6670fd624f08d5c79adfc8da5dd4a11b9213db49a3b750845e@52.178.209.125:30379
```

The user is prompted for the symmetric encryption password. The symmetric key is derived from this password. The topic will be derived from the password as well, unless it's provided by the user on the command line (which is strongly encouraged for any meaningful communication):

```
> wnode -topic=a6fcb30d -boot=enode://d25474361659861e9e651bc728a17e807a3359ca0d344afd544ed0f11a31faecaf4d74b55db53c6670fd624f08d5c79adfc8da5dd4a11b9213db49a3b750845e@52.178.209.125:30379
```

The communication is therefore established. Typing message in one console will appear on the other:

```
hello world!

1493061848 <031792461900245c6919c4b23447ef8ba43f79a2>: hello world!
```
	
The first number (1493061848) is UNIX timestamp. This format is useful for Mail Client/Server tests. The number in brackets is the ID with which the message is signed. Seeing an ID with only zeros means the message is not signed, although encrypted with the right key. Another `wnode` peer will show the same output:

```
1493061848 [031792461900245c6919c4b23447ef8ba43f79a2]: hello world!
```

Only the brackets are different, indicating that this message originated from another identity.

### Chat with Asymmetric Encryption

Using asymmetric encryption is as simple as using the `-asym` flag:

```
> wnode -topic=a6fcb30d -asym -boot=enode://d25474361659861e9e651bc728a17e807a3359ca0d344afd544ed0f11a31faecaf4d74b55db53c6670fd624f08d5c79adfc8da5dd4a11b9213db49a3b750845e@52.178.209.125:30379

my public key: 0x0405007821171295a716c9d091371e836e98a5206d5b9ce9177df90c83fc308ebae2786a9c7bff999ad83d12be08e597d4b5a5240f3bb0bc366f008b7d0908df8a 
enode://efe233263c78482111ba6c058ccc69b7a2ea3372774733def4fd5a357dfbaa67657e665078d573f11876fd2b7d75d41926976f41e257f91b486e9d36f4143c8a@[::]:42562
Whisper node started
Connected to peer.
Please enter the peer's public key:
```

First line of the output contains the key which should be passed to another peer, and vice versa. Once both clients have entered their peer's public key, the chat session is active.

### Sending and receiving files

File exchange is activated with the `fileexchange` flags. Examples here use the `-test` flag for simplicity. Assuming that the incoming messages are to be stored in `/home/tester/tmp/msg`, the resulting command line is:

```
> wnode -standalone -ip=127.0.0.1:30379 -idfile=pk1.txt -fileexchange -savedir=/home/tester/tmp/msg
enode://7d13360f5b1ddcf6947f244639113597a863abba0589d2fa5fffb2816ead0acea6211d5778a8be648e45e81ed881f4c1f5c9bbbf0e79065dfb54bcd97de3beab@127.0.0.1:30379
```

To send a file to this first `wnode`, type:

```
> wnode -test -boot=enode://7d13360f5b1ddcf6947f244639113597a863abba0589d2fa5fffb2816ead0acea6211d5778a8be648e45e81ed881f4c1f5c9bbbf0e79065dfb54bcd97de3beab@127.0.0.1:30379
```

Typing messages in the console of the second node, will cause the first one to display something like this:

```
1493124416 {624fdf6983940c7ffa8a4742f76dc78ae9775c47}: message received and saved as 'aa6f339e830c86718ddf4254038dd9fa8da6494e3f3c856af500a5aeaf0df62d' (4 bytes)
```

Messages are not displayed, but saved instead. Examine the contents of `/home/tester/tmp/msg/aa6f339e830c86718ddf4254038dd9fa8da6494e3f3c856af500a5aeaf0df62d` to confirm that the message is saved there.

It is possible to send a file directly by typing its path. For example:

```
> /home/tester/tmp/msg/aa6f339e830c86718ddf4254038dd9fa8da6494e3f3c856af500a5aeaf0df62d
```
	
Asymmetric encryption is also available in file exchange mode by providing the `asym` flag.

### Mail Server & Client

Whisper protocol allows you to exchange messages with other peers only if you are online. But what if you go offline? Will important messages be lost forever? The golang implementation of Whisper v6 has a built-in support for Mail Client/Server functionality, which allows to create very secure (and even dark) anonymous email-like system. Wnode is designed to demonstrate the viability of such project.

Mail Server and Client must have direct connection, since they exchange special kind of messages, which are not propagated any further. The reason for that is simple: if you receive the old (expired) message from the Server, and try to send it to other peers, they will recognise the message as expired, and drop connection with you.

Starting Mail Server:

	> wnode -mailserver -forwarder -standalone -test -ip=127.0.0.1:30381 -idfile=pk1.txt -dbpath=/home/tester/tmp/arj

Now start another node, connect to the Server, and send some test messages to fill the database:

	> wnode -test -boot=enode://7d13360f5b1ddcf6947f244639113597a863abba0589d2fa5fffb2816ead0acea6211d5778a8be648e45e81ed881f4c1f5c9bbbf0e79065dfb54bcd97de3beab@127.0.0.1:30381

Note the UNIX time of the messages. For example: 1493127055.
Now start the Mail Client and connect to the Server:

	> wnode -mailclient -test -boot=enode://7d13360f5b1ddcf6947f244639113597a863abba0589d2fa5fffb2816ead0acea6211d5778a8be648e45e81ed881f4c1f5c9bbbf0e79065dfb54bcd97de3beab@127.0.0.1:30381

You will be prompted to enter the time range of the archived messages you want to receive:

	> Please enter the lower limit of the time range (unix timestamp): 1493127000
	> Please enter the upper limit of the time range (unix timestamp): 1493127099
	> Please enter the topic (hexadecimal): 

You can leave the topic empty for now, in which case all the messages will be delivered, regardless of the topic.
The message should be delivered by the the Server, decrypted by the Client and displayed on the screen.
