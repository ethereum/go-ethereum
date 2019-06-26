Manifests
===============

.. _swarm-manifest:

In general manifests declare a list of strings associated with Swarm hashes. A manifest matches to exactly one hash, and it consists of a list of entries declaring the content which can be retrieved through that hash. This is demonstrated by the following example:

Let's create a directory containing the two orange papers and an html index file listing the two pdf documents.

.. code-block:: none

  $ ls -1 orange-papers/
  index.html
  smash.pdf
  sw^3.pdf

  $ cat orange-papers/index.html
  <!DOCTYPE html>
  <html lang="en">
    <head>
      <meta charset="utf-8">
    </head>
    <body>
      <ul>
        <li>
          <a href="./sw^3.pdf">Viktor Trón, Aron Fischer, Dániel Nagy A and Zsolt Felföldi, Nick Johnson: swap, swear and swindle: incentive system for swarm.</a>  May 2016
        </li>
        <li>
          <a href="./smash.pdf">Viktor Trón, Aron Fischer, Nick Johnson: smash-proof: auditable storage for swarm secured by masked audit secret hash.</a> May 2016
        </li>
      </ul>
    </body>
  </html>

We now use the ``swarm up`` command to upload the directory to Swarm to create a mini virtual site.

.. note::
   In this example we are using the public gateway through the `bzz-api` option in order to upload. The examples below assume a node running on localhost to access content. Make sure to run a local node to reproduce these examples.

.. code-block:: none

  $ swarm --recursive --defaultpath orange-papers/index.html --bzzapi http://swarm-gateways.net/ up orange-papers/ 2> up.log
  > 2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d

The returned hash is the hash of the manifest for the uploaded content (the orange-papers directory):

We now can get the manifest itself directly (instead of the files they refer to) by using the bzz-raw protocol ``bzz-raw``:

.. code-block:: none

  $ wget -O- "http://localhost:8500/bzz-raw:/2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d"

  > {
    "entries": [
      {
        "hash": "4b3a73e43ae5481960a5296a08aaae9cf466c9d5427e1eaa3b15f600373a048d",
        "contentType": "text/html; charset=utf-8"
      },
      {
        "hash": "4b3a73e43ae5481960a5296a08aaae9cf466c9d5427e1eaa3b15f600373a048d",
        "contentType": "text/html; charset=utf-8",
        "path": "index.html"
      },
      {
        "hash": "69b0a42a93825ac0407a8b0f47ccdd7655c569e80e92f3e9c63c28645df3e039",
        "contentType": "application/pdf",
        "path": "smash.pdf"
      },
      {
        "hash": "6a18222637cafb4ce692fa11df886a03e6d5e63432c53cbf7846970aa3e6fdf5",
        "contentType": "application/pdf",
        "path": "sw^3.pdf"
      }
    ]
  }

.. note::
  macOS users can install wget via homebrew (or use curl).


Manifests contain content_type information for the hashes they reference. In other contexts, where content_type is not supplied or, when you suspect the information is wrong, it is possible to specify the content_type manually in the search query. For example, the manifest itself should be `text/plain`:

.. code-block:: none

   http://localhost:8500/bzz-raw:/2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d?content_type="text/plain"

Now you can also check that the manifest hash matches the content (in fact, Swarm does this for you):

.. code-block:: none

   $ wget -O- http://localhost:8500/bzz-raw:/2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d?content_type="text/plain" > manifest.json

   $ swarm hash manifest.json
   > 2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d


Path Matching
^^^^^^^^^^^^^^

A useful feature of manifests is that we can match paths with URLs.
In some sense this makes the manifest a routing table and so the manifest acts as if it was a host.

More concretely, continuing in our example, when we request:

.. code-block:: none

  GET http://localhost:8500/bzz:/2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d/sw^3.pdf

Swarm first retrieves the document matching the manifest above. The url path ``sw^3`` is then matched against the entries. In this case a perfect match is found and the document at 6a182226... is served as a pdf.

As you can see the manifest contains 4 entries, although our directory contained only 3. The extra entry is there because of the ``--defaultpath orange-papers/index.html`` option to ``swarm up``, which associates the empty path with the file you give as its argument. This makes it possible to have a default page served when the url path is empty.
This feature essentially implements the most common webserver rewrite rules used to set the landing page of a site served when the url only contains the domain. So when you request

.. code-block:: none

  GET http://localhost:8500/bzz:/2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d/

you get served the index page (with content type ``text/html``) at ``4b3a73e43ae5481960a5296a08aaae9cf466c9d5427e1eaa3b15f600373a048d``.

Paths and directories
^^^^^^^^^^^^^^^^^^^^^

Swarm manifests don't "break" like a file system. In a file system, the directory matches at the path separator (`/` in linux) at the end of a directory name:


.. code-block:: none

  -- dirname/
  ----subdir1/
  ------subdir1file.ext
  ------subdir2file.ext
  ----subdir2/
  ------subdir2file.ext

In Swarm, path matching does not happen on a given path separator, but **on common prefixes**. Let's look at an example:
The current manifest for the ``theswarm.eth`` homepage is as follows:

.. code-block:: none

  wget -O- "http://swarm-gateways.net/bzz-raw:/theswarm.eth/ > manifest.json

  > {"entries":[{"hash":"ee55bc6844189299a44e4c06a4b7fbb6d66c90004159c67e6c6d010663233e26","path":"LICENSE","mode":420,"size":1211,"mod_time":"2018-06-12T15:36:29Z"},
              {"hash":"57fc80622275037baf4a620548ba82b284845b8862844c3f56825ae160051446","path":"README.md","mode":420,"size":96,"mod_time":"2018-06-12T15:36:29Z"},
              {"hash":"8919df964703ccc81de5aba1b688ff1a8439b4460440a64940a11e1345e453b5","path":"Swarm_files/","contentType":"application/bzz-manifest+json","mod_time":"0001-01-01T00:00:00Z"},
              {"hash":"acce5ad5180764f1fb6ae832b624f1efa6c1de9b4c77b2e6ec39f627eb2fe82c","path":"css/","contentType":"application/bzz-manifest+json","mod_time":"0001-01-01T00:00:00Z"},
              {"hash":"0a000783e31fcf0d1b01ac7d7dae0449cf09ea41731c16dc6cd15d167030a542","path":"ethersphere/orange-papers/","contentType":"application/bzz-manifest+json","mod_time":"0001-01-01T00:00:00Z"},
              {"hash":"b17868f9e5a3bf94f955780e161c07b8cd95cfd0203d2d731146746f56256e56","path":"f","contentType":"application/bzz-manifest+json","mod_time":"0001-01-01T00:00:00Z"},
              {"hash":"977055b5f06a05a8827fb42fe6d8ec97e5d7fc5a86488814a8ce89a6a10994c3","path":"i","contentType":"application/bzz-manifest+json","mod_time":"0001-01-01T00:00:00Z"},
              {"hash":"48d9624942e927d660720109b32a17f8e0400d5096c6d988429b15099e199288","path":"js/","contentType":"application/bzz-manifest+json","mod_time":"0001-01-01T00:00:00Z"},
              {"hash":"294830cee1d3e63341e4b34e5ec00707e891c9e71f619bc60c6a89d1a93a8f81","path":"talks/","contentType":"application/bzz-manifest+json","mod_time":"0001-01-01T00:00:00Z"},
              {"hash":"12e1beb28d86ed828f9c38f064402e4fac9ca7b56dab9cf59103268a62a2b35f","contentType":"text/html; charset=utf-8","mode":420,"size":31371,"mod_time":"2018-06-12T15:36:29Z"}
    ]}


Note the ``path`` for entry ``b17868...``: It is ``f``. This means, there are more than one entries for this manifest which start with an `f`, and all those entries will be retrieved by requesting the hash ``b17868...`` and through that arrive at the matching manifest entry:

.. code-block:: none

   $ wget -O- http://localhost:8500/bzz-raw:/b17868f9e5a3bf94f955780e161c07b8cd95cfd0203d2d731146746f56256e56/

   {"entries":[{"hash":"25e7859eeb7366849f3a57bb100ff9b3582caa2021f0f55fb8fce9533b6aa810","path":"avicon.ico","mode":493,"size":32038,"mod_time":"2018-06-12T15:36:29Z"},
               {"hash":"97cfd23f9e36ca07b02e92dc70de379a49be654c7ed20b3b6b793516c62a1a03","path":"onts/glyphicons-halflings-regular.","contentType":"application/bzz-manifest+json","mod_time":"0001-01-01T00:00:00Z"}
    ]}

So we can see that the ``f`` entry in the root hash resolves to a manifest containing ``avicon.ico`` and ``onts/glyphicons-halflings-regular``. The latter is interesting in itself: its ``content_type`` is ``application/bzz-manifest+json``, so it points to another manifest. Its ``path`` also does contain a path separator, but that does not result in a new manifest after the path separator like a directory (e.g. at ``onts/``). The reason is that on the file system on the hard disk, the ``fonts`` directory only contains *one* directory named ``glyphicons-halflings-regular``, thus creating a new manifest for just ``onts/`` would result in an unnecessary lookup. This general approach has been chosen to limit unnecessary lookups that would only slow down retrieval, and manifest "forks" happen in order to have the logarythmic bandwidth needed to retrieve a file in a directory with thousands of files.

When requesting ``wget -O- "http://swarm-gateways.net/bzz-raw:/theswarm.eth/favicon.ico``, Swarm will first retrieve the manifest at the root hash, match on the first ``f`` in the entry list, resolve the hash for that entry and finally resolve the hash for the ``favicon.ico`` file.

For the ``theswarm.eth`` page, the same applies to the ``i`` entry in the root hash manifest. If we look up that hash, we'll find entries for ``mages/`` (a further manifest), and ``ndex.html``, whose hash resolves to the main ``index.html`` for the web page.

Paths like ``css/`` or ``js/`` get their own manifests, just like common directories, because they contain several files.

.. note::
   If a request is issued which Swarm can not resolve unambiguosly, a ``300 "Multiplce Choices"`` HTTP status will be returned.
   In the example above, this would apply for a request for ``http://swarm-gateways.net/bzz:/theswarm.eth/i``, as it could match both ``images/`` as well as ``index.html``
