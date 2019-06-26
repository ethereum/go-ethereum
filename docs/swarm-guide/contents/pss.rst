*******************************
PSS
*******************************

:dfn:`pss` (Postal Service over Swarm) is a messaging protocol over Swarm with strong privacy features.
The pss API is exposed through a JSON RPC interface described in the `API Reference <./apireference.rst#PSS>`_,
here we explain the basic concepts and features.


.. note::

  ``pss`` is still an experimental feature and under active development and is available as of POC3 of Swarm. Expect things to change.

.. note::

  There is no CLI support for ``pss``. 


Basics
=============

With ``pss`` you can send messages to any node in the Swarm network. The messages are routed in the same manner as retrieve requests for chunks. Instead of chunk hash reference, ``pss`` messages specify a destination in the overlay address space independently of the message payload. This destination can describe a *specific node* if it is a complete overlay address or a *neighbourhood* if it is partially specified one. Up to the destination, the message is relayed through devp2p peer connections using :dfn:`forwarding kademlia` (passing messages via semi-permanent peer-to-peer TCP connections between relaying nodes using kademlia routing). Within the destination neighbourhood the message is broadcast using gossip.

Since ``pss`` messages are encrypted, ultimately *the recipient is whoever can decrypt the message*. Encryption can be done using asymmetric or symmetric encryption methods.

The message payload is dispatched to *message handlers* by the recipient nodes and dispatched to subscribers via the API.

.. important::
  ``pss`` does not guarantee message ordering (`Best-effort delivery <https://en.wikipedia.org/wiki/Best-effort_delivery>`_)
  nor message delivery (e.g. messages to offline nodes will not be cached and replayed) at the moment.

Privacy features
------------------

Thanks to end-to-end encryption, pss caters for private communication.

Due to forwarding kademlia, ``pss`` offers sender anonymity.

Using partial addressing, ``pss`` offers a sliding scale of recipient anonymity: the larger the destination neighbourhood (the smaller prefix you reveal of the intended recipient overlay address), the more difficult it is to identify the real recipient. On the other hand, since dark routing is inefficient, there is a trade-off between anonymity on the one hand and message delivery latency and bandwidth (and therefore cost) on the other. This choice is left to the application.

Forward secrecy is provided if you use the `Handshakes` module.

Usage
===========================

See the `API Reference <./apireference.rst#PSS>`_ for details.

Registering a recipient
--------------------------

Intended recipients first need to be registered with the node. This registration includes the following data:

1. ``Encryption key`` - can be a ECDSA public key for asymmetric encryption or a 32 byte symmetric key.

2. ``Topic`` - an arbitrary 4 byte word.

3. ``Address``- destination (fully or partially specified Swarm overlay address) to use for deterministic routing.

   The registration returns a key id which is used to refer to the stored key in subsequent operations.

After you associate an encryption key with an address they will be checked against any message that comes through (when sending or receiving) given it matches the topic and the destination of the message.

Sending a message
------------------

There are a few prerequisites for sending a message over ``pss``:

1. ``Encryption key id`` - id of the stored recipient's encryption key.

2. ``Topic`` - an arbitrary 4 byte word (with the exception of ``0x0000`` to be reserved for ``raw`` messages).

3. ``Message payload`` - the message data as an arbitrary byte sequence.

.. note::
  The Address that is coupled with the encryption key is used for routing the message.
  This does *not* need to be a full address; the network will route the message to the best
  of its ability with the information that is available.
  If *no* address is given (zero-length byte slice), routing is effectively deactivated,
  and the message is passed to all peers by all peers.

Upon sending the message it is encrypted and passed on from peer to peer. Any node along the route that can successfully decrypt the message is regarded as a recipient. If the destination is a neighbourhood, the message is passed around so ultimately it reaches the intended recipient which also forwards the message to their peers, recipients will continue to pass on the message to their peers, to make it harder for anyone spying on the traffic to tell where the message "ended up."

After you associate an encryption key with a destination they will be checked against any message that comes through (when sending or receiving) given it matches the topic and the address in the message.

.. important::
  When using the internal encryption methods, you MUST associate keys (whether symmetric or asymmetric) with an address space AND a topic before you will be able to send anything.

Sending a raw message
----------------------

It is also possible to send a message without using the builtin encryption. In this case no recipient registration is made, but the message is sent directly, with the following input data:

1. ``Message payload`` - the message data as an arbitrary byte sequence.

2. ``Address``- the Swarm overlay address to use for the routing.

Receiving messages
--------------------

You can subscribe to incoming messages using a topic. Since subscription needs push notifications, the supported RPC transport interfaces are websockets and IPC.

.. important::
  ``pss`` does not guarantee message ordering (`Best-effort delivery <https://en.wikipedia.org/wiki/Best-effort_delivery>`_)
  nor message delivery (e.g. messages to offline nodes will not be cached and replayed) at the moment.
  
Advanced features
==================

.. note:: This functionalities are optional features in pss. They are compiled in by default, but can be omitted by providing the appropriate build tags.

Handshakes
-----------

``pss`` provides a convenience implementation of Diffie-Hellman handshakes using ephemeral symmetric keys. Peers keep separate sets of keys for a limited amount of incoming and outgoing communications, and create and exchange new keys when the keys expire.


Protocols
-----------

A framework is also in place for making ``devp2p`` protocols available using ``pss`` connections. This feature is only available using the internal golang API, read more in the GoDocs or the codes.
