.. _updownload:

***************************
Uploading and downloading
***************************

..  contents::

Introduction
==================================
.. note:: This guide assumes you've installed the Swarm client and have a running node that listens by default on port 8500. See `Getting Started <./gettingstarted.html>`_ for details.

Arguably, uploading and downloading content is the raison d'être of Swarm. Uploading content consists of "uploading" content to your local Swarm node, followed by your local Swarm node "syncing" the resulting chunks of data with its peers in the network. Meanwhile, downloading content consists of your local Swarm node querying its peers in the network for the relevant chunks of data and then reassembling the content locally.

Uploading and downloading data can be done through the ``swarm`` command line interface (CLI) on the terminal or via the HTTP interface on `http://localhost:8500 <http://localhost:8500>`_.

Using HTTP
======================

Swarm offers an HTTP API. Thus, a simple way to upload and download files to/from Swarm is through this API.
We can use the ``curl`` `tool <https://curl.haxx.se/docs/httpscripting.html>`_ to exemplify how to interact with this API.

.. note:: Files can be uploaded in a single HTTP request, where the body is either a single file to store, a tar stream (application/x-tar) or a multipart form (multipart/form-data).

To upload a single file to your node, run this:

.. code-block:: none

  $ curl -H "Content-Type: text/plain" --data "some-data" http://localhost:8500/bzz:/

Once the file is uploaded, you will receive a hex string which will look similar to this:

.. code-block:: none

  027e57bcbae76c4b6a1c5ce589be41232498f1af86e1b1a2fc2bdffd740e9b39

This is the Swarm hash of the address string of your content inside Swarm. It is the same hash that would have been returned by using the :ref:``swarm up <swarmup>`` command.

To download a file from Swarm, you just need the file's Swarm hash. Once you have it, the process is simple. Run:

.. code-block:: none

  $ curl http://localhost:8500/bzz:/027e57bcbae76c4b6a1c5ce589be41232498f1af86e1b1a2fc2bdffd740e9b39/

The result should be your file:

.. code-block:: none

  some-data

And that's it.

.. note:: If you omit the trailing slash from the url then the request will result in a HTTP redirect. The semantically correct way to access the root path of a Swarm manifest is using the trailing slash.

Tar stream upload
------------------

Tar is a traditional unix/linux file format for packing a directory structure into a single file. Swarm provides a convenient way of using this format to make it possible to perform recursive uploads using the HTTP API.

.. code-block:: none

  # create two directories with a file in each
  $ mkdir dir1 dir2
  $ echo "some-data" > dir1/file.txt
  $ echo "some-data" > dir2/file.txt

  # create a tar archive containing the two directories (this will tar everything in the working directory)
  tar cf files.tar .

  # upload the tar archive to Swarm to create a manifest
  $ curl -H "Content-Type: application/x-tar" --data-binary @files.tar http://localhost:8500/bzz:/
  > 1e0e21894d731271e50ea2cecf60801fdc8d0b23ae33b9e808e5789346e3355e

You can then download the files using:

.. code-block:: none

  $ curl http://localhost:8500/bzz:/1e0e21894d731271e50ea2cecf60801fdc8d0b23ae33b9e808e5789346e3355e/dir1/file.txt
  > some-data

  $ curl http://localhost:8500/bzz:/1e0e21894d731271e50ea2cecf60801fdc8d0b23ae33b9e808e5789346e3355e/dir2/file.txt
  > some-data

GET requests work the same as before with the added ability to download multiple files by setting `Accept: application/x-tar`:

.. code-block:: none

  $ curl -s -H "Accept: application/x-tar" http://localhost:8500/bzz:/ccef599d1a13bed9989e424011aed2c023fce25917864cd7de38a761567410b8/ | tar t
  > dir1/file.txt
    dir2/file.txt


Multipart form upload
---------------------

.. code-block:: none

  $ curl -F 'dir1/file.txt=some-data;type=text/plain' -F 'dir2/file.txt=some-data;type=text/plain' http://localhost:8500/bzz:/
  > 9557bc9bb38d60368f5f07aae289337fcc23b4a03b12bb40a0e3e0689f76c177

  $ curl http://localhost:8500/bzz:/9557bc9bb38d60368f5f07aae289337fcc23b4a03b12bb40a0e3e0689f76c177/dir1/file.txt
  > some-data

  $ curl http://localhost:8500/bzz:/9557bc9bb38d60368f5f07aae289337fcc23b4a03b12bb40a0e3e0689f76c177/dir2/file.txt
  > some-data


Add files to an existing manifest using multipart form
------------------------------------------------------

.. code-block:: none

  $ curl -F 'dir3/file.txt=some-other-data;type=text/plain' http://localhost:8500/bzz:/9557bc9bb38d60368f5f07aae289337fcc23b4a03b12bb40a0e3e0689f76c177
  > ccef599d1a13bed9989e424011aed2c023fce25917864cd7de38a761567410b8

  $ curl http://localhost:8500/bzz:/ccef599d1a13bed9989e424011aed2c023fce25917864cd7de38a761567410b8/dir1/file.txt
  > some-data

  $ curl http://localhost:8500/bzz:/ccef599d1a13bed9989e424011aed2c023fce25917864cd7de38a761567410b8/dir3/file.txt
  > some-other-data


Upload files using a simple HTML form
-------------------------------------

.. code-block:: html

  <form method="POST" action="/bzz:/" enctype="multipart/form-data">
    <input type="file" name="dir1/file.txt">
    <input type="file" name="dir2/file.txt">
    <input type="submit" value="upload">
  </form>


Listing files
-------------

.. note:: The ``jq`` command mentioned below is a separate application that can be used to pretty-print the json data retrieved from the ``curl`` request

A `GET` request with ``bzz-list`` URL scheme returns a list of files contained under the path, grouped into common prefixes which represent directories:

.. code-block:: none

   $ curl -s http://localhost:8500/bzz-list:/ccef599d1a13bed9989e424011aed2c023fce25917864cd7de38a761567410b8/ | jq .
   > {
      "common_prefixes": [
        "dir1/",
        "dir2/",
        "dir3/"
      ]
    }

.. code-block:: none

    $ curl -s http://localhost:8500/bzz-list:/ccef599d1a13bed9989e424011aed2c023fce25917864cd7de38a761567410b8/dir1/ | jq .
    > {
      "entries": [
        {
          "path": "dir1/file.txt",
          "contentType": "text/plain",
          "size": 9,
          "mod_time": "2017-03-12T15:19:55.112597383Z",
          "hash": "94f78a45c7897957809544aa6d68aa7ad35df695713895953b885aca274bd955"
        }
      ]
    }

Setting ``Accept: text/html`` returns the list as a browsable HTML document.


Using CLI
=====================

.. _swarmup:

Uploading a file to your local Swarm node
------------------------------------------

.. note:: Once a file is uploaded to your local Swarm node, your node will `sync` the chunks of data with other nodes on the network. Thus, the file will eventually be available on the network even when your original node goes offline.

The basic command for uploading to your local node is ``swarm up FILE``. For example, let's create a file called example.md and issue the following command to upload the file example.md file to your local Swarm node.

.. code-block:: none
  
  $ echo "this is an example" > example.md
  $ swarm up example.md
  > d1f25a870a7bb7e5d526a7623338e4e9b8399e76df8b634020d11d969594f24a

The hash returned is the hash of a :ref:`swarm manifest <swarm-manifest>`. This manifest is a JSON file that contains the ``example.md`` file as its only entry. Both the primary content and the manifest are uploaded by default.

After uploading, you can access this example.md file from Swarm by pointing your browser to:

.. code-block:: none

  $ http://localhost:8500/bzz:/d1f25a870a7bb7e5d526a7623338e4e9b8399e76df8b634020d11d969594f24a/

The manifest makes sure you could retrieve the file with the correct MIME type.

You can encrypt your file using the ``--encrypt`` flag. See the :ref:`Encryption` section for details.


Suppressing automatic manifest creation
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
You may wish to prevent a manifest from being created alongside with your content and only upload the raw content. You might want to include it in a custom index, or handle it as a data-blob known and used only by a certain application that knows its MIME type. For this you can set ``--manifest=false``:

.. code-block:: none

  $ swarm --manifest=false up FILE
  > 7149075b7f485411e5cc7bb2d9b7c86b3f9f80fb16a3ba84f5dc6654ac3f8ceb

This option suppresses automatic manifest upload. It uploads the content as-is.
However, if you wish to retrieve this file, the browser can not be told unambiguously what that file represents.
In the context, the hash ``7149075b7f485411e5cc7bb2d9b7c86b3f9f80fb16a3ba84f5dc6654ac3f8ceb`` does not refer to a manifest. Therefore, any attempt to retrieve it using the ``bzz:/`` scheme will result in a ``404 Not Found`` error. In order to access this file, you would have to use the :ref:`bzz-raw` scheme.


Downloading a single file
----------------------------

To download single files, use the ``swarm down`` command.
Single files can be downloaded in the following different manners. The following examples assume ``<hash>`` resolves into a single-file manifest:

.. code-block:: none

  $ swarm down bzz:/<hash>            #downloads the file at <hash> to the current working directory
  $ swarm down bzz:/<hash> file.tmp   #downloads the file at <hash> as ``file.tmp`` in the current working dir
  $ swarm down bzz:/<hash> dir1/      #downloads the file at <hash> to ``dir1/``

You can also specify a custom proxy with `--bzzapi`:

.. code-block:: none

  $ swarm --bzzapi http://localhost:8500 down bzz:/<hash>            #downloads the file at <hash> to the current working directory using the localhost node


Downloading a single file from a multi-entry manifest can be done with (``<hash>`` resolves into a multi-entry manifest):

.. code-block:: none

  $ swarm down bzz:/<hash>/index.html            #downloads index.html to the current working directory
  $ swarm down bzz:/<hash>/index.html file.tmp   #downloads index.html as file.tmp in the current working directory
  $ swarm down bzz:/<hash>/index.html dir1/      #downloads index.html to dir1/

..If you try to download from a multi-entry manifest without specifying the file, you will get a `got too many matches for this path` error. You will need to specify a `--recursive` flag (see below).

Uploading to a remote Swarm node
-----------------------------------
You can upload to a remote Swarm node using the ``--bzzapi`` flag.
For example, you can use one of the public gateways as a proxy, in which case you can upload to Swarm without even running a node.


.. code-block:: none

  $ swarm --bzzapi https://swarm-gateways.net up example.md

.. note:: This gateway currently only accepts uploads of limited size. In future, the ability to upload to this gateways is likely to disappear entirely.


Uploading a directory
-----------------------

Uploading directories is achieved with the ``--recursive`` flag.

.. code-block:: none

  $ swarm --recursive up /path/to/directory
  > ab90f84c912915c2a300a94ec5bef6fc0747d1fbaf86d769b3eed1c836733a30

The returned hash refers to a root manifest referencing all the files in the directory.

Directory with default entry
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

It is possible to declare a default entry in a manifest. In the example above, if ``index.html`` is declared as the default, then a request for a resource with an empty path will show the contents of the file ``/index.html``

.. code-block:: none

  $ swarm --defaultpath /path/to/directory/index.html --recursive up /path/to/directory
  > ef6fc0747d1fbaf86d769b3eed1c836733a30ab90f84c912915c2a300a94ec5b

You can now access index.html at

.. code-block:: none

  $ http://localhost:8500/bzz:/ef6fc0747d1fbaf86d769b3eed1c836733a30ab90f84c912915c2a300a94ec5b/

and also at

.. code-block:: none

  $ http://localhost:8500/bzz:/ef6fc0747d1fbaf86d769b3eed1c836733a30ab90f84c912915c2a300a94ec5b/index.html

This is especially useful when the hash (in this case ``ef6fc0747d1fbaf86d769b3eed1c836733a30ab90f84c912915c2a300a94ec5b``) is given a registered name like ``mysite.eth`` in the `Ethereum Name Service <./ens.html>`_. In this case the lookup would be even simpler:

.. code-block:: none

  http://localhost:8500/bzz:/mysite.eth/

.. note:: You can toggle automatic default entry detection with the ``SWARM_AUTO_DEFAULTPATH`` environment variable. You can do so by a simple ``$ export SWARM_AUTO_DEFAULTPATH=true``. This will tell Swarm to automatically look for ``<uploaded directory>/index.html`` file and set it as the default manifest entry (in the case it exists).  

Downloading a directory
--------------------------

To download a directory, use the ``swarm down --recursive`` command.
Directories can be downloaded in the following different manners. The following examples assume <hash> resolves into a multi-entry manifest:

.. code-block:: none

  $ swarm down --recursive bzz:/<hash>            #downloads the directory at <hash> to the current working directory
  $ swarm down --recursive bzz:/<hash> dir1/      #downloads the file at <hash> to dir1/

Similarly as with a single file, you can also specify a custom proxy with ``--bzzapi``:

.. code-block:: none

  $ swarm --bzzapi http://localhost:8500 down --recursive bzz:/<hash> #note the flag ordering

.. important :: Watch out for the order of arguments in directory upload/download: it's ``swarm --recursive up`` and ``swarm down --recursive``.

Adding entries to a manifest
-------------------------------
The command for modifying manifests is ``swarm manifest``.

To add an entry to a manifest, use the command:

.. code-block:: none

  $ swarm manifest add <manifest-hash> <path> <hash> [content-type]

To remove an entry from a manifest, use the command:

.. code-block:: none

  $ swarm manifest remove <manifest-hash> <path>

To modify the hash of an entry in a manifest, use the command:

.. code-block:: none

  $ swarm manifest update <manifest-hash> <path> <new-hash>

Reference table
-----------------

+------------------------------------------+------------------------------------------------------------------------+
| **upload**                               | ``swarm up <file>``                                                    |
+------------------------------------------+------------------------------------------------------------------------+
| ~ dir                                    | ``swarm --recursive up <dir>``                                         |
+------------------------------------------+------------------------------------------------------------------------+
| ~ dir w/ default entry (here: index.html)| ``swarm --defaultpath <dir>/index.html --recursive up <dir>``          |
+------------------------------------------+------------------------------------------------------------------------+ 
| ~ w/o manifest                           | ``swarm --manifest=false up``                                          |
+------------------------------------------+------------------------------------------------------------------------+
| ~ to remote node                         | ``swarm --bzzapi https://swarm-gateways.net up``                       |
+------------------------------------------+------------------------------------------------------------------------+
| ~ with encryption                        | ``swarm up --encrypt``                                                 |
+------------------------------------------+------------------------------------------------------------------------+
| **download**                             | ``swarm down bzz:/<hash>``                                             |
+------------------------------------------+------------------------------------------------------------------------+
| ~ dir                                    | ``swarm down --recursive bzz:/<hash>``                                 |
+------------------------------------------+------------------------------------------------------------------------+
| ~ as file                                | ``swarm down bzz:/<hash> file.tmp``                                    |
+------------------------------------------+------------------------------------------------------------------------+
| ~ into dir                               | ``swarm down bzz:/<hash> dir/``                                        |
+------------------------------------------+------------------------------------------------------------------------+
| ~ w/ custom proxy                        | ``swarm down --bzzapi http://<proxy address> down bzz:/<hash>``        |
+------------------------------------------+------------------------------------------------------------------------+
| **manifest**                             |                                                                        |
+------------------------------------------+------------------------------------------------------------------------+
| add ~                                    | ``swarm manifest add <manifest-hash> <path> <hash> [content-type]``    |
+------------------------------------------+------------------------------------------------------------------------+
| remove ~                                 | ``swarm manifest remove <manifest-hash> <path>``                       |
+------------------------------------------+------------------------------------------------------------------------+
| update ~                                 | ``swarm manifest update <manifest-hash> <path> <new-hash>``            |
+------------------------------------------+------------------------------------------------------------------------+

Up- and downloading in the CLI: example usage
----------------------------------

.. tabs::

  .. group-tab:: Up/downloading

    Let's create a dummy file and upload it to Swarm:

    .. code-block:: none

      $ echo "this is a test" > myfile.md
      $ swarm up myfile.md
      > <reference hash>

    We can download it using the ``bzz:/`` scheme and give it a name.

    .. code-block:: none

      $ swarm down bzz:/<reference hash> iwantmyfileback.md
      $ cat iwantmyfileback.md
      > this is a test

    We can also ``curl`` it using the HTTP API.

    .. code-block:: none

      $ curl http://localhost:8500/bzz:/<reference hash>/
      > this is a test

    We can use the ``bzz-raw`` scheme to see the manifest of the upload.

    .. code-block:: none

      $ curl http://localhost:8500/bzz-raw:/<reference hash>/

    This returns the manifest:

    .. code-block:: none

      {
        "entries": [
          {
            "hash": "<file hash>",
            "path": "myfile.md",
            "contentType": "text/markdown; charset=utf-8",
            "mode": 420,
            "size": 15,
            "mod_time": "<timestamp>"
          }
        ]
      }

  .. group-tab:: Up/down as is

    We can upload the file as-is:

    .. code-block:: none

      $ echo "this is a test" > myfile.md
      $ swarm --manifest=false up myfile.md
      > <as-is reference hash>

    We can retrieve it using the ``bzz-raw`` scheme in the HTTP API.

    .. code-block:: none

      $ curl http://localhost:8500/bzz-raw:/<as-is reference hash>/
      > this is a test

  .. group-tab:: Manipulate manifests

    Let's create a directory with a dummy file, and upload the directory to swarm.

    .. code-block:: none 

      $ mkdir dir
      $ echo "this is a test" > dir/dummyfile.md
      $ swarm --recursive up dir
      > <dir hash>

    We can look at the manifest using ``bzz-raw`` and the HTTP API.

    .. code-block:: none 
    
      $ curl http://localhost:8500/bzz-raw:/<dir hash>/

    It will look something like this:

    .. code-block:: none

      {
        "entries": [
          {
            "hash": "<file hash>",
            "path": "dummyfile.md",
            "contentType": "text/markdown; charset=utf-8",
            "mode": 420,
            "size": 15,
            "mod_time": "2018-11-11T16:52:07+01:00"
          }
        ]
      }

    We can remove the file from the manifest using ``manifest remove``.

    .. code-block:: none

      $ swarm manifest remove <dir hash> "dummyfile.md"
      > <new dir hash>

    When we check the new dir hash, we notice that it's empty -- as it should be.

    Let's put the file back in there.

    .. code-block:: none

      $ swarm up dir/dummyfile.md
      > <individual file hash>
      $ swarm manifest add <new dir hash> "dummyfileagain.md" <individual file hash>
      > <new dir hash 2>

    We can check the manifest under <new dir hash 2> to see that the file is back there.