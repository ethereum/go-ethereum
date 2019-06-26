Access Control 
===============

Swarm supports restricting access to content through several access control strategies:

- Password protection - where a number of undisclosed parties can access content using a shared secret ``(pass, act)``

- Selective access using `Elliptic Curve <https://en.wikipedia.org/wiki/Elliptic-curve_cryptography>`_ key-pairs:

    - For an undisclosed party - where only one grantee can access the content ``(pk)``

    - For a number of undisclosed parties - where every grantee can access the content ``(act)``

**Creating** access control for content is currently supported only through CLI usage.

**Accessing** restricted content is available through CLI and HTTP. When accessing content which is restricted by a password `HTTP Basic access authentication <https://en.wikipedia.org/wiki/Basic_access_authentication>`_ can be used out-of-the-box.

.. important:: When accessing content which is restricted to certain EC keys - the node which exposes the HTTP proxy that is queried must be started with the granted private key as its ``bzzaccount`` CLI parameter.

Password protection 
-------------------

The simplest type of credential is a passphrase. In typical use cases, the
passphrase is distributed by off-band means, with adequate security measures. 
Any user that knows the passphrase can access the content.

When using password protection, a given content reference (e.g.: a given Swarm manifest address or, alternatively, 
a Mutable Resource address) is encrypted using `scrypt <https://en.wikipedia.org/wiki/Scrypt>`_
with a given passphrase and a random salt. 
The encrypted reference and the salt are then embedded into an unencrypted manifest which can be freely
distributed but only accessed by undisclosed parties that posses knowledge of the passphrase.

Password protection can also be used for selective access when using the ``act`` strategy - similarly to granting access to a certain EC key access can be also given to a party identified by a password. In fact, one could also create an ``act`` manifest that solely grants access to grantees through passwords, without the need to know their public keys.

Example usage:

.. important:: Restricting access to content on Swarm is a 2-step process - you first upload your content, then wrap the reference with an access control manifest. **We recommend that you always upload your content with encryption enabled**. In the following examples we will refer the uploaded content hash as ``reference hash``

First, we create a simple test file. We upload it to Swarm (with encryption).

.. code-block:: none

  $ echo "testfile" > mytest.txt
  $ swarm up --encrypt mytest.txt
  > <reference hash>

Then, for the sake of this example, we create a file with our password in it.

.. code-block:: none

  $ echo "mypassword" > mypassword.txt

This password will protect the access-controlled content that we upload. We can refer to this password using the `--password` flag. The password file should contain the password in plaintext. 

The ``swarm access`` command sets a new password using the ``new pass`` argument. It expects you to input the password file and the uploaded Swarm content hash you'd like to limit access to.

.. code-block:: bash

  $ swarm access new pass --password mypassword.txt <reference hash>
  > <reference of access controlled manifest>

The returned hash is the hash of the access controlled manifest. 

When requesting this hash through the HTTP gateway you should receive an ``HTTP Unauthorized 401`` error:

.. code-block:: bash

  $ curl http://localhost:8500/bzz:/<reference of access controlled manifest>/
  > Code: 401
  > Message: cant decrypt - forbidden
  > Timestamp: XXX

You can retrieve the content in three ways:

1. The same request should make an authentication dialog pop-up in the browser. You could then input the password needed and the content should correctly appear. (Leave the username empty.)
2. Requesting the same hash with HTTP basic authentication would return the content too. ``curl`` needs you to input a username as well as a password, but the former can be an arbitrary string (here, it's ``x``).

.. code-block:: bash

  $ curl http://x:mypassword@localhost:8500/bzz:/<reference of access controlled manifest>/

3. You can also use ``swarm down`` with the ``--password`` flag.  

.. code-block:: bash

  $ swarm  --password mypassword.txt down bzz:/<reference of access controlled manifest>/ mytest2.txt
  $ cat mytest2.txt
  > testfile

Selective access using EC keys
-------------------------------

A more sophisticated type of credential is an `Elliptic Curve <https://en.wikipedia.org/wiki/Elliptic-curve_cryptography>`_
private key, identical to those used throughout Ethereum for accessing accounts. 

In order to obtain the content reference, an
`Elliptic-curve Diffie–Hellman <https://en.wikipedia.org/wiki/Elliptic-curve_Diffie%E2%80%93Hellman>`_ `(ECDH)` 
key agreement needs to be performed between a provided EC public key (that of the content publisher) 
and the authorized key, after which the undisclosed authorized party can decrypt the reference to the 
access controlled content.

Whether using access control to disclose content to a single party (by using the ``pk`` strategy) or to 
multiple parties (using the ``act`` strategy), a third unauthorized party cannot find out the identity 
of the authorized parties.
The third party can, however, know the number of undisclosed grantees to the content. 
This, however, can be mitigated by adding bogus grantee keys while using the ``act`` strategy 
in cases where masking the number of grantees is necessary. This is not the case when using the ``pk`` strategy, as it as
by definition an agreement between two parties and only two parties (the publisher and the grantee).

.. important::
  Accessing content which is access controlled is enabled only when using a `local` Swarm node (e.g. running on `localhost`) in order to keep
  your data, passwords and encryption keys safe. This is enforced through an in-code guard.

.. danger:: 
  **NEVER (EVER!) use an external gateway to upload or download access controlled content as you will be putting your privacy at risk!
  You have been fairly warned!**

**Protecting content with Elliptic curve keys (single grantee):**

The ``pk`` strategy requires a ``bzzaccount`` to encrypt with. The most comfortable option in this case would be the same ``bzzaccount`` you normally start your Swarm node with - this will allow you to access your content seamlessly through that node at any given point in time.

Grantee public keys are expected to be in an *secp256 compressed* form - 66 characters long string (an example would be ``02e6f8d5e28faaa899744972bb847b6eb805a160494690c9ee7197ae9f619181db``). Comments and other characters are not allowed.

.. code-block:: bash

	$ swarm --bzzaccount <your account> access new pk --grant-key <your public key> <reference hash>
	> <reference of access controlled manifest>

The returned hash ``4b964a75ab19db960c274058695ca4ae21b8e19f03ddf1be482ba3ad3c5b9f9b`` is the hash of the access controlled manifest. 

The only way to fetch the access controlled content in this case would be to request the hash through one of the nodes that were granted access and/or posses the granted private key (and that the requesting node has been started with the appropriate ``bzzaccount`` that is associated with the relevant key) - either the local node that was used to upload the content or the node which was granted access through its public key.

**Protecting content with Elliptic curve keys and passwords (multiple grantees):**

The ``act`` strategy also requires a ``bzzaccount`` to encrypt with. The most comfortable option in this case would be the same ``bzzaccount`` you normally start your Swarm node with - this will allow you to access your content seamlessly through that node at any given point in time

.. note:: the ``act`` strategy expects a grantee public-key list and/or a list of permitted passwords to be communicated to the CLI. This is done using the ``--grant-keys`` flag and/or the ``--password`` flag. Grantee public keys are expected to be in an *secp256 compressed* form - 66 characters long string (e.g. ``02e6f8d5e28faaa899744972bb847b6eb805a160494690c9ee7197ae9f619181db``). Each grantee should appear in a separate line. Passwords are also expected to be line-separated. Comments and other characters are not allowed.

.. code-block:: bash

	swarm --bzzaccount 2f1cd699b0bf461dcfbf0098ad8f5587b038f0f1 access new act --grant-keys /path/to/public-keys/file --password /path/to/passwords/file  <reference hash>
	4b964a75ab19db960c274058695ca4ae21b8e19f03ddf1be482ba3ad3c5b9f9b

The returned hash ``4b964a75ab19db960c274058695ca4ae21b8e19f03ddf1be482ba3ad3c5b9f9b`` is the hash of the access controlled manifest. 

As with the ``pk`` strategy - the only way to fetch the access controlled content in this case would be to request the hash through one of the nodes that were granted access and/or posses the granted private key (and that the requesting node has been started with the appropriate ``bzzaccount`` that is associated with the relevant key) - either the local node that was used to upload the content or one of the nodes which were granted access through their public keys.

HTTP usage
----------

Accessing restricted content on Swarm through the HTTP API is, as mentioned, limited to your local node
due to security considerations.
Whenever requesting a restricted resource without the proper credentials via the HTTP proxy, the Swarm node will respond 
with an ``HTTP 401 Unauthorized`` response code.

*When accessing password protected content:*

When accessing a resource protected by a passphrase without the appropriate credentials the browser will 
receive an ``HTTP 401 Unauthorized`` response and will show a pop-up dialog asking for a username and password.
For the sake of decrypting the content - only the password input in the dialog matters and the username field can be left blank.

The credentials for accessing content protected by a password can be provided in the initial request in the form of:
``http://x:<password>@localhost:8500/bzz:/<hash or ens name>`` (``curl`` needs you to input a username as well as a password, but the former can be an arbitrary string (here, it's ``x``).)

.. important:: Access controlled content should be accessed through the ``bzz://`` protocol

*When accessing EC key protected content:*

When accessing a resource protected by EC keys, the node that requests the content will try to decrypt the restricted
content reference using its **own** EC key which is associated with the current `bzz account` that 
the node was started with (see the ``--bzzaccount`` flag). If the node's key is granted access - the content will be
decrypted and displayed, otherwise - an ``HTTP 401 Unauthorized`` error will be returned by the node.

Access control in the CLI: example usage
-----------------------------------------

.. tabs::

  .. group-tab:: Passwords

    First, we create a simple test file. We upload it to Swarm using encryption.
    
    .. code-block:: none
    
      $ echo "testfile" > mytest.txt
      $ swarm up  --encrypt mytest.txt
      > <reference hash>
  
    Then, we define a password file and use it to create an access-controlled manifest.
  
    .. code-block:: none
    
      $ echo "mypassword" > mypassword.txt
      $ swarm access new pass --password mypassword.txt <reference hash>
      > <reference of access controlled manifest>
    
    We can create a passwords file with one password per line in plaintext (``password1`` is probably not a very good password).
    
    .. code-block:: bash
    
      $ for i in {1..3}; do echo -e password$i; done > mypasswords.txt
      $ cat mypasswords.txt
      > password1
      > password2
      > password3
    
    Then, we point to this list while wrapping our manifest.
    
    .. code-block:: bash
    
      $ swarm access new act --password mypasswords.txt <reference hash>
      > <reference of access controlled manifest>
    
    We can access the returned manifest using any of the passwords in the password list.
    
    .. code-block:: bash
    
      $ echo password1 > password1.txt  
      $ swarm --password1.txt down bzz:/<reference of access controlled manifest>
    
    We can also `curl` it.
    
    .. code-block:: bash
    
      $ curl http://:password1@localhost:8500/bzz:/<reference of access controlled manifest>/
  
  .. group-tab:: Elliptic curve keys

    1. ``pk`` strategy

    First, we create a simple test file. We upload it to Swarm using encryption.
    
      .. code-block:: none
    
        $ echo "testfile" > mytest.txt
        $ swarm up --encrypt mytest.txt
        > <reference hash>

    Then, we draw an EC key pair and use the public key to create the access-controlled manifest.

      .. code-block:: none

        $ swarm access new pk --grant-key <public key> <reference hash>
        > <reference of access controlled manifest>

    We can retrieve the access-controlled manifest via a node that has the private key. You can add a private key using ``geth`` (see `here <https://github.com/ethereum/go-ethereum/wiki/Managing-your-accounts>`_).

      .. code-block:: none

        $ swarm --bzzaccount <address of node with granted private key> down bzz:/<reference of access controlled manifest> out.txt
        $ cat out.txt
        > "testfile"

    2. ``act`` strategy

    We can also supply a list of public keys to create the access-controlled manifest.

      .. code-block:: none

        $ swarm access new act --grant-keys <public key list> <reference hash>
        > <reference of access controlled manifest>

    Again, only nodes that possess the private key will have access to the content.
    
    .. code-block:: none

        $ swarm --bzzaccount <address of node with a granted private key> down bzz:/<reference of access controlled manifest> out.txt
        $ cat out.txt
        > "testfile"    