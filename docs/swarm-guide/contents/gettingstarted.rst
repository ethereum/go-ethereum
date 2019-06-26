.. _Getting Started:

******************************
Getting started
******************************

The first thing to do is to start up your Swarm node and connect it to the Swarm network.

Running Swarm
=============

To start a basic Swarm node you must have both ``geth`` and ``swarm`` installed on your machine. You can find the relevant instructions in the `Installation and Updates <./installation.html>`_  section. ``geth`` is the go-ethereum client, you can read up on it in the `Ethereum Homestead documentation <http://ethdocs.org/en/latest/ethereum-clients/go-ethereum/index.html>`_.

To start Swarm you need an Ethereum account. You can create a new account in ``geth`` by running the following command:

.. code-block:: none

  $ geth account new

You will be prompted for a password:

.. code-block:: none

  Your new account is locked with a password. Please give a password. Do not forget this password.
  Passphrase:
  Repeat passphrase:

Once you have specified the password, the output will be the Ethereum address representing that account. For example:

.. code-block:: none

  Address: {2f1cd699b0bf461dcfbf0098ad8f5587b038f0f1}

Using this account, connect to Swarm with

.. code-block:: none

  $ swarm --bzzaccount <your-account-here>
  # in our example
  $ swarm --bzzaccount 2f1cd699b0bf461dcfbf0098ad8f5587b038f0f1

(You should replace ``2f1cd699b0bf461dcfbf0098ad8f5587b038f0f1`` with your account address key).

.. important::

  **Remember your password.** There is no *forgot my password* option for ``swarm`` and ``geth``.  

Verifying that your local Swarm node is running
-----------------------------------------------

When running, ``swarm`` is accessible through an HTTP API on port 8500. Confirm that it is up and running by pointing your browser to http://localhost:8500 (You should see a Swarm search box.)

Interacting with Swarm
======================

.. _3.2:

The easiest way to access Swarm through the command line, or through the `Geth JavaScript Console <http://ethdocs.org/en/latest/account-management.html>`_ by attaching the console to a running swarm node. ``$BZZKEY$`` refers to your account address key.

.. tabs::

    .. group-tab:: Linux

      .. code-block:: none

        $ swarm --bzzaccount $BZZKEY

      And, in a new terminal window:    

      .. code-block:: none

        $ geth attach $HOME/.ethereum/bzzd.ipc

    .. group-tab:: macOS

      .. code-block:: none

        $ swarm --bzzaccount $BZZKEY

      And, in a new terminal window:    

      .. code-block:: none

        $ geth attach $HOME/Library/Ethereum/bzzd.ipc

    .. group-tab:: Windows

      .. code-block:: none

        $ swarm --bzzaccount $BZZKEY

      And, in a new terminal window:

      .. code-block:: none

        $ geth attach \\.\pipe\bzzd.ipc


Swarm is fully compatible with Geth Console commands. For example, you can list your peers using ``admin.peers``, add a peer using ``admin.addPeer``, and so on.

You can use Swarm with CLI flags and environment variables. See a full list in the `Configuration <./configuration.html>`_ .

.. _connect-ens:

How do I enable ENS name resolution?
=====================================

The `Ethereum Name Service <http://ens.readthedocs.io/en/latest/introduction.html>`_ (ENS) is the Ethereum equivalent of DNS in the classic web. It is based on a suite of smart contracts running on the *Ethereum mainnet*. 

In order to use **ENS** to resolve names to swarm content hashes, ``swarm`` has to connect to a ``geth`` instance that is connected to the *Ethereum mainnet*. This is done using the ``--ens-api`` flag.

First you must start your geth node and establish connection with Ethereum main network with the following command:

.. code-block:: none

  $ geth

for a full geth node, or

.. code-block:: none

  $ geth --syncmode=light

for light client mode.

.. note::

  **Syncing might take a while.** When you use the light mode, you don't have to sync the node before it can be used to answer ENS queries. However, please note that light mode is still an experimental feature.

After the connection is established, open another terminal window and connect to Swarm:

.. tabs::

    .. group-tab:: Linux

      .. code-block:: none

        $ swarm --ens-api $HOME/.ethereum/geth.ipc \
        --bzzaccount $BZZKEY

    .. group-tab:: macOS

      .. code-block:: none

        $ swarm --ens-api $HOME/Library/Ethereum/geth.ipc \
        --bzzaccount $BZZKEY

    .. group-tab:: Windows

      .. code-block:: none

        $ swarm --ens-api \\.\pipe\geth.ipc \
        --bzzaccount $BZZKEY


Verify that this was successful by pointing your browser to http://localhost:8500/bzz:/theswarm.eth/

Using Swarm together with the testnet ENS
------------------------------------------

It is also possible to use the Ropsten ENS test registrar for name resolution instead of the Ethereum main .eth ENS on mainnet.

Run a geth node connected to the Ropsten testnet

.. code-block:: none

  $ geth --testnet

Then launch the ``swarm``; connecting it to the geth node (``--ens-api``).

.. tabs::

    .. group-tab:: Linux

      .. code-block:: none

        $ swarm --ens-api $HOME/.ethereum/geth/testnet/geth.ipc \
        --bzzaccount $BZZKEY

    .. group-tab:: macOS

      .. code-block:: none

        $ swarm --ens-api $HOME/Library/Ethereum/geth/testnet/geth.ipc \
        --bzzaccount $BZZKEY

    .. group-tab:: Windows

      .. code-block:: none

        $ swarm --ens-api \\.\pipe\geth.ipc \
        --bzzaccount $BZZKEY


Swarm will automatically use the ENS deployed on Ropsten.

For other ethereum blockchains and other deployments of the ENS contracts, you can specify the contract addresses manually. For example the following command:

.. code-block:: none

  $ swarm --ens-api eth:<contract 1>@/home/user/.ethereum/geth.ipc \
           --ens-api test:<contract 2>@ws:<address 1> \
           --ens-api <contract 3>@ws:<address 2>

Will use the ``geth.ipc`` to resolve ``.eth`` names using the contract at address ``<contract 1>`` and it will use ``ws:<address 1>`` to resolve ``.test`` names using the contract at address ``<contract 2>``. For all other names it will use the ENS contract at address ``<contract 3>`` on ``ws:<address 2>``.

Using an external ENS source
----------------------------

.. important::

  Take care when using external sources of information. By doing so you are trusting someone else to be truthful. Using an external ENS source may make you vulnerable to man-in-the-middle attacks. It is only recommended for test and development environments.

Maintaining a fully synced Ethereum node comes with certain hardware and bandwidth constraints, and can be tricky to achieve. Also, light client mode, where syncing is not necessary, is still experimental.

An alternative solution for development purposes is to connect to an external node that you trust, and that offers the necessary functionality through HTTP.

If the external node is running on IP 12.34.56.78 port 8545, the command would be:

.. code-block:: none

  $ swarm --ens-api http://12.34.45.78:8545

You can also use ``https``. But keep in mind that Swarm *does not validate the certificate*.


Alternative modes
=================

Below are examples on ways to run ``swarm`` beyond just the default network. You can instruct Swarm using the geth command line interface or use the geth javascript console.

Swarm in singleton mode (no peers)
------------------------------------

If you **don't** want your swarm node to connect to any existing networks, you can provide it with a custom network identifier using ``--bzznetworkid`` with a random large number.


.. tabs::

    .. group-tab:: Linux

      .. code-block:: none

        $ swarm --bzzaccount $BZZKEY \
        --datadir $HOME/.ethereum \
        --ens-api $HOME/.ethereum/geth.ipc \
        --bzznetworkid <random number between 15 and 256>

    .. group-tab:: macOS

      .. code-block:: none

        $ swarm --bzzaccount $BZZKEY \
        --datadir $HOME/Library/Ethereum/ \
        --ens-api $HOME/Library/Ethereum/geth.ipc \
        --bzznetworkid <random number between 15 and 256>

    .. group-tab:: Windows

      .. code-block:: none

        $ swarm --bzzaccount $BZZKEY \
        --datadir %HOMEPATH%\AppData\Roaming\Ethereum \
        --ens-api \\.\pipe\geth.ipc \
        --bzznetworkid <random number between 15 and 256>

Adding enodes manually
------------------------

By default, Swarm will automatically seek out peers in the network.

Additionally you can manually start off the connection process by adding one or more peers using the ``admin.addPeer`` console command.

.. tabs::

    .. group-tab:: Linux

      .. code-block:: none

        $ geth --exec='admin.addPeer("ENODE")' attach $HOME/.ethereum/bzzd.ipc

    .. group-tab:: macOS

      .. code-block:: none

        $ geth --exec='admin.addPeer("ENODE")' attach $HOME/Library/Ethereum/bzzd.ipc

    .. group-tab:: Windows

      .. code-block:: none

        $ geth --exec='admin.addPeer("ENODE")' attach \\.\pipe\bzzd.ipc

(You can also do this in the Geth Console, as seen in Section 3.2_.)

.. note::

  When you stop a node, all peer connections will be saved. When you start again, the node will try to reconnect to those peers automatically.

Where ENODE is the enode record of a swarm node. Such a record looks like the following:

.. code-block:: none

  enode://01f7728a1ba53fc263bcfbc2acacc07f08358657070e17536b2845d98d1741ec2af00718c79827dfdbecf5cfcd77965824421508cc9095f378eb2b2156eb79fa@1.2.3.4:30399

The enode of your swarm node can be accessed using ``geth`` connected to ``bzzd.ipc``

.. tabs::

    .. group-tab:: Linux

      .. code-block:: none

        $ geth --exec "admin.nodeInfo.enode" attach $HOME/.ethereum/bzzd.ipc

    .. group-tab:: macOS

      .. code-block:: none

        $ geth --exec "admin.nodeInfo.enode" attach $HOME/Library/Ethereum/bzzd.ipc

    .. group-tab:: Windows

      .. code-block:: none

        $ geth --exec "admin.nodeInfo.enode" attach \\.\pipe\bzzd.ipc


.. note::
  Note how ``geth`` is used for two different purposes here: You use it to run an Ethereum Mainnet node for ENS lookups. But you also use it to "attach" to the Swarm node to send commands to it.

Connecting to the public Swarm cluster
--------------------------------------

By default Swarm connects to the public Swarm testnet operated by the Ethereum Foundation and other contributors.

The nodes the team maintains function as a free-to-use public access gateway to Swarm, so that users can experiment with Swarm without the need to run a local node. To download data through the gateway use the ``https://swarm-gateways.net/bzz:/<address>/`` URL.

Metrics reporting
------------------

Swarm uses the `go-metrics` library for metrics collection. You can set your node to collect metrics and push them to an influxdb database (called `metrics` by default) with the default settings. Tracing is also supported. An example of a default configuration is given below:

.. code-block:: none

  $ swarm --bzzaccount <bzzkey> \
  --debug \
  --metrics \
  --metrics.influxdb.export \
  --metrics.influxdb.endpoint "http://localhost:8086" \
  --metrics.influxdb.username "user" \
  --metrics.influxdb.password "pass" \
  --metrics.influxdb.database "metrics" \
  --metrics.influxdb.host.tag "localhost" \
  --verbosity 4 \
  --tracing \
  --tracing.endpoint=jaeger:6831 \
  --tracing.svc myswarm
  
