## Wnode

Wnode (whisper node) is a command-line diagnostic tool. It does not have a nice user interface, because its main purpose is diagnostic, and it's meant to be very light-weight rather than beautiful. Wnode might be used for different purposes, including:

- running a standalone bootstrap node, in order to set up a private whisper network
- connecting to particular node for debugging or other purposes
- engaging in command-line chat with another peer in order to test its functionality
- sending and receiving text messages
- sending and receiving files
- running fully functional mail server
- testing functionality of mail client

#### Usage

	> wnode [flags/arguments]
	
For example:

	> wnode -forwarder -standalone -ip=127.0.0.1:30381 -idfile=config.txt -mailserver -dbpath=tmp/db

#### Flags

-asym: use asymmetric encryption in the chat

-fileexchange: file exchange mode (send and receive files instead of text messages)

-forwarder: forwarder mode (only forward; neither send nor decrypt messages)

-mailserver: mail server mode (delivers expired messages on demand)

-mailclient: request expired messages from the mail server

-standalone: don't actively connect to any peers, wait for incoming connections instead

-test: use of predefined parameters for diagnostics, including passwords

-generatekey: generate a new private key (ECIES) and exit

#### Arguments

In case of missing arguments wnode will either use default value or ask you to provide them. For security reasons, you can not provide passwords in command-line arguments. Only in test mode hardcoded passwords are used.

-verbosity:
Verbosity level of logging. Int value between 0 and 5. Default value 1. For example: -verbosity=5.

-ttl:
Time-to-live for messages in seconds. Default value 30. For example: -ttl=20.

-maxsize:
Maximum allowed message size in bytes. 

-work:
Maximum time in seconds allowed to spend on proof of work in order to achieve the target (set by 'pow' argument).

-pow:
PoW target for normal messages in float format (e.g. -pow=2.7).

-mspow:
PoW requirement for Mail Server request.

-ip:
IP address and port of this node (e.g. 127.0.0.1:30303).

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

### Scenarios & Examples

For simplicity, in these examples we assume that we only use wnode to communicate with another wnode.

#### Start a bootstrap node for test network

	> wnode -standalone -forwarder -ip=127.0.0.1:30379

result output:

	my public key: 0x040ef7acd60781c336c52056b3782f7eae45be2063e591ac6b78472dc27ba770010bde445ffd2f3623ad656f3859e00d11ef518df4916c4d4e258c60b15f34c682	enode://15454fc65bbf0031155f4eee83fa732f1454c314e9f78ade9cba4d4a098d29edbf5431764ee65b200169025c3f900cacc3348a000dda7a8a0d9643d0b7618712@127.0.0.1:30379
	Bootstrap Whisper node started

After the bootstrap node has started, another local node can connect to it, using the resulting enode:

	> wnode -test -boot=enode://15454fc65bbf0031155f4eee83fa732f1454c314e9f78ade9cba4d4a098d29edbf5431764ee65b200169025c3f900cacc3348a000dda7a8a0d9643d0b7618712@127.0.0.1:30379

result output:

	............................
	Whisper node started
	Connected to peer.
	............................
	
Now, if you restart you bootstrap node, its enode will be different, because new ID will be randomly generated. If you need to repeat the tests multiple times, it will be extremely boring to copy and paste new enode every time you restart. Instead, you can load ID from file using 'idfile' argument. 

Generating ID:

	> wnode -generatekey

result: 

	c74ea2702eb32f523acb118649998e1c8b5690cf0a14bffda7e87b411db3499a

Then save it to file:

	> echo c74ea2702eb32f523acb118649998e1c8b5690cf0a14bffda7e87b411db3499a > pk1.txt
	
Then start the bootstrap node with persistent ID:

	> wnode -forwarder -standalone -ip=127.0.0.1:30379 -idfile=pk1.txt
	
result:

	my public key: 0x04be81a00a90f5c21ead8887eaa254b3f7a37e06f8f2d776dcc46954a228bc50c6fb6dfd155f7e44e6fef9b62fdf6dad041759b864d2cbe4089b6f5c16a817ff46	enode://7d13360f5b1ddcf6947f244639113597a863abba0589d2fa5fffb2816ead0acea6211d5778a8be648e45e81ed881f4c1f5c9bbbf0e79065dfb54bcd97de3beab@127.0.0.1:30379
	Filter is configured for the topic: 5a4ea131 
	Bootstrap Whisper node started

Now you can always use the same command to connect to your bootstrap node:

	> wnode -test -boot=enode://7d13360f5b1ddcf6947f244639113597a863abba0589d2fa5fffb2816ead0acea6211d5778a8be648e45e81ed881f4c1f5c9bbbf0e79065dfb54bcd97de3beab@127.0.0.1:30379
	
Please note that ID file is stored unencrypted. It should be used only for test purposes.

Of course, two local whisper nodes are only useful for the most basic tests. 
In order to set up a bootstrap node on a server with dedicated IP address, you need to specify its IP explicitly:

	> wnode -forwarder -standalone -ip=52.178.211.103:30379

#### Chat

Now we will start a chat between two or more nodes sharing the same password, using symmetric encryption. One of the nodes should be started with 'standalone' flag, and another must connect to the first one. It is easy to do on the same machine or on a dedicated server. But what if two peers are behind distinct NAT? In that case, you need a third bootstrap node on a dedicated server, which both peers can connect to. At the time of writing we have out test node with the following enode:
enode://d25474361659861e9e651bc728a17e807a3359ca0d344afd544ed0f11a31faecaf4d74b55db53c6670fd624f08d5c79adfc8da5dd4a11b9213db49a3b750845e@52.178.209.125:30379,
to which both peers can connect with the following command:

	> wnode -boot=enode://d25474361659861e9e651bc728a17e807a3359ca0d344afd544ed0f11a31faecaf4d74b55db53c6670fd624f08d5c79adfc8da5dd4a11b9213db49a3b750845e@52.178.209.125:30379

Then you will be prompted to enter the password for symmetric encryption. From this password symmetric key will be derived. The topic will be derived from the password as well, unless you provide your own (which is strongly encouraged for any meaningful communication):

	> wnode -topic=a6fcb30d -boot=enode://d25474361659861e9e651bc728a17e807a3359ca0d344afd544ed0f11a31faecaf4d74b55db53c6670fd624f08d5c79adfc8da5dd4a11b9213db49a3b750845e@52.178.209.125:30379

Now you can type text messages:

	hello world!

	1493061848 <031792461900245c6919c4b23447ef8ba43f79a2>: hello world!
	
You will see your own message successfully decrypted and printed on the screen. The first number (1493061848) is UNIX time in seconds. This format is useful for Mail Client/Server tests. The number in brackets is ID with which the message is signed. In this case -- your own ID. If you see only zeros there, it means the message is not signed, although encrypted with the right key. Another wnode peer will show the same output:

	1493061848 [031792461900245c6919c4b23447ef8ba43f79a2]: hello world!

Almost the same, only the brackets are different, indicating that this is not its own message -- originated from somebody else.

#### Chat with Asymmetric Encryption

Connect both peers to the same bootstrap node again, but this time with 'asym' flag:

	> wnode -topic=a6fcb30d -asym -boot=enode://d25474361659861e9e651bc728a17e807a3359ca0d344afd544ed0f11a31faecaf4d74b55db53c6670fd624f08d5c79adfc8da5dd4a11b9213db49a3b750845e@52.178.209.125:30379

result:

	my public key: 0x0405007821171295a716c9d091371e836e98a5206d5b9ce9177df90c83fc308ebae2786a9c7bff999ad83d12be08e597d4b5a5240f3bb0bc366f008b7d0908df8a 
	enode://efe233263c78482111ba6c058ccc69b7a2ea3372774733def4fd5a357dfbaa67657e665078d573f11876fd2b7d75d41926976f41e257f91b486e9d36f4143c8a@[::]:42562
	Whisper node started
	Connected to peer.
	Please enter the peer's public key:

First line of the output contains the key which should be passed to anther peer, and vice versa. Then you can proceed with the chat as in the previous case.

#### Sending and receiving files

Start wnode with 'fileexchange' flag, and 'test' flag for simplicity. Suppose we want to store the incoming messages in the directory /home/tester/tmp/msg. In that case:

	> wnode -standalone -ip=127.0.0.1:30379 -idfile=pk1.txt -fileexchange -savedir=/home/tester/tmp/msg

Now, start another wnode and connect to the first one:

	> wnode -test -boot=enode://7d13360f5b1ddcf6947f244639113597a863abba0589d2fa5fffb2816ead0acea6211d5778a8be648e45e81ed881f4c1f5c9bbbf0e79065dfb54bcd97de3beab@127.0.0.1:30379

After you will type and send messages from the second node, you will see the first one to display something like this:

	1493124416 {624fdf6983940c7ffa8a4742f76dc78ae9775c47}: message received and saved as 'aa6f339e830c86718ddf4254038dd9fa8da6494e3f3c856af500a5aeaf0df62d' (4 bytes)

As you see, messages are not displayed, but saved instead. Now you can open the file /home/tester/tmp/msg/aa6f339e830c86718ddf4254038dd9fa8da6494e3f3c856af500a5aeaf0df62d and examine its contents.

If you want to send a file from the 'fileexchange' peer, just type the file name. For example:

	> /home/tester/tmp/msg/aa6f339e830c86718ddf4254038dd9fa8da6494e3f3c856af500a5aeaf0df62d
	
Another peer should receive the message, decrypt and display it on the screen. 
If you want to use your own password instead of hardcoded one, just call wnode without 'test' flag.
Of course, you can also switch to asymmetric encryption by providing 'asym' flag.

#### Mail Server & Client

Whisper protocol allows you to exchange messages with other peers only if you are online. But what if you go offline? Will important messages be lost forever? The golang implementation of Whisper v5 has a built-in support for Mail Client/Server functionality, which allows to create very secure (and even dark) anonymous email-like system. Wnode is designed to demonstrate the viability of such project.

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
