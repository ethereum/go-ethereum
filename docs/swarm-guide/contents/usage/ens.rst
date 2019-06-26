.. _Ethereum Name Service:

Using ENS names
================

.. note:: In order to `resolve` ENS names, your Swarm node has to be connected to an Ethereum blockchain (mainnet, or testnet). See `Getting Started <./gettingstarted.html#connect-ens>`_ for instructions. This section explains how you can register your content to your ENS name.

`ENS <http://ens.readthedocs.io/en/latest/introduction.html>`_ is the system that Swarm uses to permit content to be referred to by a human-readable name, such as "theswarm.eth". It operates analogously to the DNS system, translating human-readable names into machine identifiers - in this case, the Swarm hash of the content you're referring to. By registering a name and setting it to resolve to the content hash of the root manifest of your site, users can access your site via a URL such as ``bzz://theswarm.eth/``.

.. note:: Currently The `bzz` scheme is not supported in major browsers such as Chrome, Firefox or Safari. If you want to access the `bzz` scheme through these browsers, currently you have to either use an HTTP gateway, such as https://swarm-gateways.net/bzz:/theswarm.eth/ or use a browser which supports the `bzz` scheme, such as Mist <https://github.com/ethereum/mist>.

Suppose we upload a directory to Swarm containing (among other things) the file ``example.pdf``.

.. code-block:: none

  $ swarm --recursive up /path/to/dir
  >2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d

If we register the root hash as the ``content`` for ``theswarm.eth``, then we can access the pdf at

.. code-block:: none

  bzz://theswarm.eth/example.pdf

if we are using a Swarm-enabled browser, or at

.. code-block:: none

  http://localhost:8500/bzz:/theswarm.eth/example.pdf

via a local gateway. We will get served the same content as with:

.. code-block:: none

  http://localhost:8500/bzz:/2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d/example.pdf

Please refer to the `official ENS documentation <http://ens.readthedocs.io/en/latest/introduction.html>`_ for the full details on how to register content hashes to ENS.

In short, the steps you must take are:

1. Register an ENS name.
2. Associate a resolver with that name.
3. Register the Swarm hash with the resolver as the ``content``.

We recommend using https://manager.ens.domains/. This will make it easy for you to:

- Associate the default resolver with your name
- Register a Swarm hash.

.. note:: When you register a Swarm hash with https://manager.ens.domains/ you MUST prefix the hash with 0x. For example 0x2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d

Overview of ENS (video)
-----------------------

Nick Johnson on the Ethereum Name System

.. raw:: html

  <iframe width="560" height="315" src="https://www.youtube.com/embed/pLDDbCZXvTE" frameborder="0" allow="autoplay; encrypted-media" allowfullscreen></iframe>
