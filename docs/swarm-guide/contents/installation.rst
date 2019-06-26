*************************
Installation and Updates
*************************

Swarm is part of the Ethereum stack, the reference implementation is currently at POC3 (proof of concept 3), or version 0.3.x


Swarm runs on all major platforms (Linux, macOS, Windows, Raspberry Pi, Android, iOS).

Swarm was written in golang and requires the go-ethereum client **geth** to run.

..  note::
  The swarm package has not been extensively tested on platforms other than Linux and macOS.

Installing Swarm on Ubuntu via PPA
==================================

The simplest way to install Swarm on **Ubuntu distributions** is via the built in launchpad PPAs (Personal Package Archives). We provide a single PPA repository that contains our stable releases for Ubuntu versions trusty, xenial, bionic and cosmic.

To enable our launchpad repository please run:

.. code-block:: shell

  $ sudo apt-get install software-properties-common
  $ sudo add-apt-repository -y ppa:ethereum/ethereum

After that you can install the stable version of Swarm:

.. code-block:: shell

  $ sudo apt-get update
  $ sudo apt-get install ethereum-swarm

Setting up Swarm in Docker
=============================

You can run Swarm in a Docker container. The official Swarm Docker image including documentation on how to run it can be found on `Github <https://github.com/ethersphere/swarm-docker/>`_ or pulled from `Docker <https://hub.docker.com/r/ethdevops/swarm/>`_.

You can run it with optional arguments, e.g.:

.. code-block:: shell

  $ docker run -e PASSWORD=<password> -t ethdevops/swarm:latest --debug --verbosity 4

In order to up/download, you need to expose the HTTP api port (here: to localhost:8501) and set the HTTP address:

.. code-block:: shell

  $ docker run -p 8501:8500/tcp -e PASSWORD=<password> -t ethdevops/swarm:latest  --httpaddr=0.0.0.0 --debug --verbosity 4

In this example, you can use ``swarm --bzzapi http://localhost:8501 up testfile.md`` to upload ``testfile.md`` to swarm using the Docker node, and you can get it back e.g. with ``curl http://localhost:8501/bzz:/<hash>``.

Note that if you want to use a pprof HTTP server, you need to expose the ports and set the address (with ``--pprofaddr=0.0.0.0``) too.

In order to attach a Geth Javascript console, you need to mount a data directory from a volume:

.. code-block:: shell

  $ docker run -p 8501:8500/tcp -e PASSWORD=<password> -e DATADIR=/data -v /tmp/hostdata:/data -t-t ethdevops/swarm:latest --httpaddr=0.0.0.0 --debug --verbosity 4

Then, you can attach the console with:

.. code-block:: shell

  $ docker exec -it swarm1 /geth attach /data/bzzd.ipc

You can also open a terminal session inside the container:

.. code-block:: shell

  $ docker exec -it swarm1 /bin/sh

Installing Swarm from source
=============================

The Swarm source code for can be found on https://github.com/ethersphere/swarm

Prerequisites: Go and Git
--------------------------

Building the Swarm binary requires the following packages:

* go: https://golang.org
* git: http://git.org


Grab the relevant prerequisites and build from source.

.. tabs::

   .. tab:: Ubuntu / Debian

      .. code-block:: shell

         $ sudo apt install git

         $ sudo add-apt-repository ppa:gophers/archive
         $ sudo apt-get update
         $ sudo apt-get install golang-1.11-go

         // Note that golang-1.11-go puts binaries in /usr/lib/go-1.11/bin. If you want them on your PATH, you need to make that change yourself.

         $ export PATH=/usr/lib/go-1.11/bin:$PATH

   .. tab:: Archlinux

      .. code-block:: shell

         $ pacman -S git go

   .. tab:: Generic Linux

      The latest version of Go can be found at https://golang.org/dl/

      To install it, download the tar.gz file for your architecture and unpack it to ``/usr/local``

   .. tab:: macOS

      .. code-block:: shell

        $ brew install go git

   .. tab:: Windows

      Take a look `here <https://medium.freecodecamp.org/setting-up-go-programming-language-on-windows-f02c8c14e2f>`_ at installing go and git and preparing your go environment under Windows.

Configuring the Go environment
-------------------------------

You should then prepare your Go environment.

.. tabs::

    .. group-tab:: Linux

      .. code-block:: shell

        $ mkdir $HOME/go
        $ echo 'export GOPATH=$HOME/go' >> ~/.bashrc
        $ echo 'export PATH=$GOPATH/bin:$PATH' >> ~/.bashrc
        $ source ~/.bashrc

    .. group-tab:: macOS

      .. code-block:: shell

        $ mkdir $HOME/go
        $ echo 'export GOPATH=$HOME/go' >> $HOME/.bash_profile
        $ echo 'export PATH=$GOPATH/bin:$PATH' >> $HOME/.bash_profile
        $ source $HOME/.bash_profile

Download and install Geth
----------------------------------------

Once all prerequisites are met, download and install Geth from https://github.com/ethereum/go-ethereum


Compiling and installing Swarm
----------------------------------------

Once all prerequisites are met, and you have ``geth`` on your system, clone the Swarm git repo and build from source:

.. code-block:: shell

  $ git clone https://github.com/ethersphere/swarm
  $ cd swarm
  $ make swarm

Alternatively you could also use the Go tooling and download and compile Swarm from `master` via:


.. code-block:: shell

  $ go get -d github.com/ethersphere/swarm
  $ go install github.com/ethersphere/swarm/cmd/swarm

You can now run ``swarm`` to start your Swarm node.
Let's check if the installation of ``swarm`` was successful:

.. code-block:: none

  swarm version

If your ``PATH`` is not set and the ``swarm`` command cannot be found, try:

  .. code-block:: shell

    $ $GOPATH/bin/swarm version

This should return some relevant information. For example:

.. code-block:: shell

  Swarm
  Version: 0.3
  Network Id: 0
  Go Version: go1.10.1
  OS: linux
  GOPATH=/home/user/go
  GOROOT=/usr/local/go

Updating your client
---------------------

To update your client simply download the newest source code and recompile.
