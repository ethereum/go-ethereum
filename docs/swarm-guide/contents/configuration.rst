******************************
Configuration
******************************

.. _configuration:

Command line options for swarm
====================================

The ``swarm`` executable supports the following configuration options:

* Configuration file
* Environment variables
* Command line

Options provided via command line override options from the environment variables, which will override options in the config file. If an option is not explicitly provided, a default will be chosen.

In order to keep the set of flags and variables manageable, only a subset of all available configuration options are available via command line and environment variables. Some are only available through a TOML configuration file.

.. note:: Swarm reuses code from ethereum, specifically some p2p networking protocol and other common parts. To this end, it accepts a number of environment variables which are actually from the ``geth`` environment. Refer to the geth documentation for reference on these flags.

This is the list of flags inherited from ``geth``:

.. code-block:: none

  --identity
  --bootnodes
  --datadir
  --keystore
  --port
  --nodiscover
  --v5disc
  --netrestrict
  --nodekey
  --nodekeyhex
  --maxpeers
  --nat
  --ipcdisable
  --ipcpath
  --password

Config File
=============

.. note:: ``swarm`` can be executed with the ``dumpconfig`` command, which prints a default configuration to STDOUT, and thus can be redirected to a file as a template for the config file.

A TOML configuration file is organized in sections. The below list of available configuration options is organized according to these sections. The sections correspond to `Go` modules, so need to be respected in order for file configuration to work properly. See `<https://github.com/naoina/toml>`_ for the TOML parser and encoder library for Golang, and `<https://github.com/toml-lang/toml>`_ for further information on TOML.

To run Swarm with a config file, use:

.. code-block:: shell

  $ swarm --config /path/to/config/file.toml

General configuration parameters
================================

.. csv-table::
   :header: "Config file", "Command-line flag", "Environment variable", "Default value", "Description"
   :widths: 10, 5, 5, 15, 55

   "n/a","--config","n/a","n/a","Path to config file in TOML format"
   "n/a","--bzzapi","n/a","http://127.0.0.1:8500","Swarm HTTP endpoint"
   "BootNodes","--bootnodes","SWARM_BOOTNODES","","Boot nodes"
   "BzzAccount","--bzzaccount","SWARM_ACCOUNT", "","Swarm account key"
   "BzzKey","n/a","n/a", "n/a","Swarm node base address (:math:`hash(PublicKey)hash(PublicKey))`. This is used to decide storage based on radius and routing by kademlia."
   "Contract","--chequebook","SWARM_CHEQUEBOOK_ADDR","0x0","Swap chequebook contract address"
   "Cors","--corsdomain","SWARM_CORS", "","Domain on which to send Access-Control-Allow-Origin header (multiple domains can be supplied separated by a ',')"
   "n/a","--debug","n/a","n/a","Prepends log messages with call-site location (file and line number)"
   "n/a","--defaultpath","n/a","n/a","path to file served for empty url path (none)"
   "n/a","--delivery-skip-check","SWARM_DELIVERY_SKIP_CHECK","false","Skip chunk delivery check (default false)"
   "EnsApi","--ens-api","SWARM_ENS_API","<$GETH_DATADIR>/geth.ipc","Ethereum Name Service API address"
   "EnsRoot","--ens-addr","SWARM_ENS_ADDR", "ens.TestNetAddress","Ethereum Name Service contract address"
   "ListenAddr","--httpaddr","SWARM_LISTEN_ADDR", "127.0.0.1","Swarm listen address"
   "n/a","--manifest value","n/a","true","Automatic manifest upload (default true)"
   "n/a","--mime value","n/a","n/a","Force mime type on upload"
   "NetworkId","--bzznetworkid","SWARM_NETWORK_ID","3","Network ID"
   "Path","--datadir","GETH_DATADIR","<$GETH_DATADIR>/swarm","Path to the geth configuration directory"
   "Port","--bzzport","SWARM_PORT", "8500","Port to run the http proxy server"
   "PublicKey","n/a","n/a", "n/a","Public key of swarm base account"
   "n/a","--recursive","n/a", "false","Upload directories recursively (default false)"
   "n/a","--stdin","","n/a","Reads data to be uploaded from stdin"
   "n/a","--store.path value","SWARM_STORE_PATH","<$GETH_ENV_DIR>/swarm/bzz-<$BZZ_KEY>/chunks","Path to leveldb chunk DB"
   "n/a","--store.size value","SWARM_STORE_CAPACITY","5000000","Number of chunks (5M is roughly 20-25GB) (default 5000000)]"
   "n/a","--store.cache.size value","SWARM_STORE_CACHE_CAPACITY","5000","Number of recent chunks cached in memory (default 5000)"            
   "n/a","--sync-update-delay value","SWARM_ENV_SYNC_UPDATE_DELAY","","Duration for sync subscriptions update after no new peers are added (default 15s)"            
   "SwapApi","--swap-api","SWARM_SWAP_API","","URL of the Ethereum API provider to use to settle SWAP payments"
   "SwapEnabled","--swap","SWARM_SWAP_ENABLE","false","Enable SWAP"
   "SyncDisabled","--nosync","SWARM_ENV_SYNC_DISABLE","false","Disable Swarm node synchronization"
   "n/a","--verbosity value","n/a","3","Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail"
   "n/a","--ws","n/a","false","Enable the WS-RPC server"
   "n/a","--wsaddr value","n/a","localhost","WS-RPC server listening interface"
   "n/a","--wsport value","n/a","8546","WS-RPC server listening port"
   "n/a","--wsapi value","n/a","n/a","API's offered over the WS-RPC interface"
   "n/a","--wsorigins value","n/a","n/a","Origins from which to accept websockets requests"
   "n/a","n/a","SWARM_AUTO_DEFAULTPATH","false","Toggle automatic manifest default path on recursive uploads (looks for index.html)"