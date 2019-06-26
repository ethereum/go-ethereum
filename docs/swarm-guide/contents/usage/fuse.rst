
FUSE
======================


Another way of interacting with Swarm is by mounting it as a local filesystem using `FUSE <https://en.wikipedia.org/wiki/Filesystem_in_Userspace>`_ (Filesystem in Userspace). There are three IPC API's which help in doing this.

.. note:: FUSE needs to be installed on your Operating System for these commands to work. Windows is not supported by FUSE, so these command will work only in Linux, Mac OS and FreeBSD. For installation instruction for your OS, see "Installing FUSE" section below.


Installing FUSE
----------------

1. Linux (Ubuntu)

.. code-block:: none

	$ sudo apt-get install fuse
	$ sudo modprobe fuse
	$ sudo chown <username>:<groupname> /etc/fuse.conf
	$ sudo chown <username>:<groupname> /dev/fuse

2. Mac OS

   Either install the latest package from https://osxfuse.github.io/ or use brew as below

.. code-block:: none

	$ brew update
	$ brew install caskroom/cask/brew-cask
	$ brew cask install osxfuse


CLI Usage
-----------

The Swarm CLI now integrates commands to make FUSE usage easier and streamlined.

.. note:: When using FUSE from the CLI, we assume you are running a local Swarm node on your machine. The FUSE commands attach to the running node through `bzzd.ipc`

Mount
^^^^^^^^

One use case to mount a Swarm hash via FUSE is a file sharing feature accessible via your local file system.
Files uploaded to Swarm are then transparently accessible via your local file system, just as if they were stored locally.

To mount a Swarm resource, first upload some content to Swarm using the ``swarm up <resource>`` command.
You can also upload a complete folder using ``swarm --recursive up <directory>``.
Once you get the returned manifest hash, use it to mount the manifest to a mount point
(the mount point should exist on your hard drive):

.. code-block:: none

	$ swarm fs mount <manifest-hash> <mount-point>


For example:

.. code-block:: none

	$ swarm fs mount <manifest-hash> /home/user/swarmmount


Your running Swarm node terminal output should show something similar to the following in case the command returned successfuly:

.. code-block:: none

	Attempting to mount /path/to/mount/point
	Serving 6e4642148d0a1ea60e36931513f3ed6daf3deb5e499dcf256fa629fbc22cf247 at /path/to/mount/point
	Now serving swarm FUSE FS                manifest=6e4642148d0a1ea60e36931513f3ed6daf3deb5e499dcf256fa629fbc22cf247 mountpoint=/path/to/mount/point

You may get a "Fatal: had an error calling the RPC endpoint while mounting: context deadline exceeded" error if it takes too long to retrieve the content.

In your OS, via terminal or file browser, you now should be able to access the contents of the Swarm hash at ``/path/to/mount/point``, i.e. ``ls /home/user/swarmmount``


Access
^^^^^^^^
Through your terminal or file browser, you can interact with your new mount as if it was a local directory. Thus you can add, remove, edit, create files and directories just as on a local directory. Every such action will interact with Swarm, taking effect on the Swarm distributed storage. Every such action also will result **in a new hash** for your mounted directory. If you would unmount and remount the same directory with the previous hash, your changes would seem to have been lost (effectively you are just mounting the previous version). While you change the current mount, this happens under the hood and your mount remains up-to-date.

Unmount
^^^^^^^^
To unmount a ``swarmfs`` mount, either use the List Mounts command below, or use a known mount point:

.. code-block:: none

	$ swarm fs unmount <mount-point>
	> 41e422e6daf2f4b32cd59dc6a296cce2f8cce1de9f7c7172e9d0fc4c68a3987a

The returned hash is the latest manifest version that was mounted.
You can use this hash to remount the latest version with the most recent changes.


List Mounts
^^^^^^^^^^^^^^^^^^
To see all existing swarmfs mount points, use the List Mounts command:

.. code-block:: none

  $ swarm fs list


Example Output:

.. code-block:: none

	Found 1 swarmfs mount(s):
	0:
		Mount point: /path/to/mount/point
		Latest Manifest: 6e4642148d0a1ea60e36931513f3ed6daf3deb5e499dcf256fa629fbc22cf247
		Start Manifest: 6e4642148d0a1ea60e36931513f3ed6daf3deb5e499dcf256fa629fbc22cf247

