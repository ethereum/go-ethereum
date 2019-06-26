.. _Encryption:

Encryption
===========

Introduced in POC 0.3, symmetric encryption is now readily available to be used with the ``swarm up`` upload command.
The encryption mechanism is meant to protect your information and make the chunked data unreadable to any handling Swarm node.

Swarm uses `Counter mode encryption <https://en.wikipedia.org/wiki/Block_cipher_mode_of_operation#Counter_(CTR)>`_ to encrypt and decrypt content. When you upload content to Swarm, the uploaded data is split into 4 KB chunks. These chunks will all be encoded with a separate randomly generated encryption key. The encryption happens on your local Swarm node, unencrypted data is not shared with other nodes. The reference of a single chunk (and the whole content) will be the concatenation of the hash of encoded data and the decryption key. This means the reference will be longer than the standard unencrypted Swarm reference (64 bytes instead of 32 bytes).

When your node syncs the encrypted chunks of your content with other nodes, it does not share the full references (or the decryption keys in any way) with the other nodes. This means that other nodes will not be able to access your original data, moreover they will not be able to detect whether the synchronized chunks are encrypted or not.

When your data is retrieved it will only get decrypted on your local Swarm node. During the whole retrieval process the chunks traverse the network in their encrypted form, and none of the participating peers are able to decrypt them. They are only decrypted and assembled on the Swarm node you use for the download.

More info about how we handle encryption at Swarm can be found `here <https://github.com/ethersphere/swarm/wiki/Symmetric-Encryption-for-Swarm-Content>`_.

.. note::
  Swarm currently supports both encrypted and unencrypted ``swarm up`` commands through usage of the ``--encrypt`` flag.
  This might change in the future as we will refine and make Swarm a safer network.

.. important::
  The encryption feature is non-deterministic (due to a random key generated on every upload request) and users of the API should not rely on the result being idempotent; thus uploading the same content twice to Swarm with encryption enabled will not result in the same reference.


Example usage:

First, we create a simple test file.

.. code-block:: none

  $ echo "testfile" > mytest.txt

We upload the test file **without** encryption, 

.. code-block:: none

  $ swarm up mytest.txt
  > <file reference>

and **with** encryption. 

.. code-block:: none

  $ swarm up --encrypt mytest.txt
  > <encrypted reference>

Note that the reference of the encrypted upload is **longer** than that of the unencrypted upload. Note also that, because of the random encryption key, repeating the encrypted upload results in a different reference:

.. code-block:: none

  $ swarm up --encrypt mytest.txt
  <another encrypted reference>

