.. _BZZ URL schemes:

BZZ URL schemes
=======================

Swarm offers 6 distinct URL schemes:

bzz
-----

The bzz scheme assumes that the domain part of the url points to a manifest. When retrieving the asset addressed by the URL, the manifest entries are matched against the URL path. The entry with the longest matching path is retrieved and served with the content type specified in the corresponding manifest entry.

Example:

.. code-block:: none

    GET http://localhost:8500/bzz:/2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d/readme.md

returns a readme.md file if the manifest at the given hash address contains such an entry.

.. code-block:: none

    $ ls
    readme.md
    $ swarm --recursive up .
    c4c81dbce3835846e47a83df549e4cad399c6a81cbf83234274b87d49f5f9020
    $ curl http://localhost:8500/bzz-raw:/c4c81dbce3835846e47a83df549e4cad399c6a81cbf83234274b87d49f5f9020/readme.md
    ## Hello Swarm!

    Swarm is awesome%

If the manifest does not contain an file at ``readme.md`` itself, but it does contain multiple entries to which the URL could be resolved, e.g. in the example above, the manifest has entries for ``readme.md.1`` and ``readme.md.2``, the API returns an HTTP response "300 Multiple Choices", indicating that the request could not be unambiguously resolved. A list of available entries is returned via HTTP or JSON.

.. code-block:: none

    $ ls
    readme.md.1 readme.md.2
    $ swarm --recursive up .
    679bde3ccb6fb911db96a0ea1586c04899c6c0cc6d3426e9ee361137b270a463
    $ curl -H "Accept:application/json" http://localhost:8500/bzz:/679bde3ccb6fb911db96a0ea1586c04899c6c0cc6d3426e9ee361137b270a463/readme.md
    {"Msg":"\u003ca href='/bzz:/679bde3ccb6fb911db96a0ea1586c04899c6c0cc6d3426e9ee361137b270a463/readme.md.1'\u003ereadme.md.1\u003c/a\u003e\u003cbr/\u003e\u003ca href='/bzz:/679bde3ccb6fb911db96a0ea1586c04899c6c0cc6d3426e9ee361137b270a463/readme.md.2'\u003ereadme.md.2\u003c/a\u003e\u003cbr/\u003e","Code":300,"Timestamp":"Fri, 15 Jun 2018 14:48:42 CEST","Details":""}
    $ curl -H "Accept:application/json" http://localhost:8500/bzz:/679bde3ccb6fb911db96a0ea1586c04899c6c0cc6d3426e9ee361137b270a463/readme.md | jq
    {
        "Msg": "<a href='/bzz:/679bde3ccb6fb911db96a0ea1586c04899c6c0cc6d3426e9ee361137b270a463/readme.md.1'>readme.md.1</a><br/><a href='/bzz:/679bde3ccb6fb911db96a0ea1586c04899c6c0cc6d3426e9ee361137b270a463/readme.md.2'>readme.md.2</a><br/>",
        "Code": 300,
        "Timestamp": "Fri, 15 Jun 2018 14:49:02 CEST",
        "Details": ""
    }

``bzz`` scheme also accepts POST requests to upload content and create manifest for them in one go:

.. code-block:: none

    $ curl -H "Content-Type: text/plain" --data-binary "some-data" http://localhost:8500/bzz:/
    635d13a547d3252839e9e68ac6446b58ae974f4f59648fe063b07c248494c7b2%
    $ curl http://localhost:8500/bzz:/635d13a547d3252839e9e68ac6446b58ae974f4f59648fe063b07c248494c7b2/
    some-data%
    $ curl -H "Accept:application/json" http://localhost:8500/bzz-raw:/635d13a547d3252839e9e68ac6446b58ae974f4f59648fe063b07c248494c7b2/ | jq .
    {
        "entries": [
            {
                "hash": "379f234c04ed1a18722e4c76b5029ff6e21867186c4dfc101be4f1dd9a879d98",
                "contentType": "text/plain",
                "mode": 420,
                "size": 9,
                "mod_time": "2018-06-15T15:46:28.835066044+02:00"
            }
        ]
    }

.. _bzz-raw:

bzz-raw
-------------

.. code-block:: none

    GET http://localhost:8500/bzz-raw:/2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d


When responding to GET requests with the bzz-raw scheme, Swarm does not assume that the hash resolves to a manifest. Instead it just serves the asset referenced by the hash directly. So if the hash actually resolves to a manifest, it returns the raw manifest content itself.

E.g. continuing the example in the ``bzz`` section above with ``readme.md.1`` and ``readme.md.2`` in the manifest:

.. code-block:: none

    $ curl http://localhost:8500/bzz-raw:/679bde3ccb6fb911db96a0ea1586c04899c6c0cc6d3426e9ee361137b270a463/ | jq
    {
        "entries": [
            {
            "hash": "efc6d4a7d7f0846973a321d1702c0c478a20f72519516ef230b63baa3da18c22",
            "path": "readme.md.",
            "contentType": "application/bzz-manifest+json",
            "mod_time": "0001-01-01T00:00:00Z"
            }
        ]
        }
    $ curl http://localhost:8500/bzz-raw:/efc6d4a7d7f0846973a321d1702c0c478a20f72519516ef230b63baa3da18c22/ | jq
    {
        "entries": [
            {
                "hash": "d0675100bc4580a0ad890b5d6f06310c0705d4ab1e796cfa1a8c597840f9793f",
                "path": "1",
                "mode": 420,
                "size": 33,
                "mod_time": "2018-06-15T14:21:32+02:00"
            },
            {
                "hash": "f97cf36ac0dd7178c098f3661cd0402fcc711ff62b67df9893d29f1db35adac6",
                "path": "2",
                "mode": 420,
                "size": 35,
                "mod_time": "2018-06-15T14:42:06+02:00"
            }
        ]
        }

The ``content_type`` query parameter can be supplied to specify the MIME type you are requesting, otherwise content is served as an octet-stream per default. For instance if you have a pdf document (not the manifest wrapping it) at hash ``6a182226...`` then the following url will properly serve it.

.. code-block:: none

    GET http://localhost:8500/bzz-raw:/6a18222637cafb4ce692fa11df886a03e6d5e63432c53cbf7846970aa3e6fdf5?content_type=application/pdf

``bzz-raw`` also supports POST requests to upload content to Swarm, the response is the hash of the uploaded content:

.. code-block:: none

    $ curl --data-binary "some-data" http://localhost:8500/bzz-raw:/
    379f234c04ed1a18722e4c76b5029ff6e21867186c4dfc101be4f1dd9a879d98%
    $ curl http://localhost:8500/bzz-raw:/379f234c04ed1a18722e4c76b5029ff6e21867186c4dfc101be4f1dd9a879d98/
    some-data%

bzz-list
-------------

.. code-block:: none

    GET http://localhost:8500/bzz-list:/2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d/path

Returns a list of all files contained in <manifest> under <path> grouped into common prefixes using ``/`` as a delimiter. If no path is supplied, all files in manifest are returned. The response is a JSON-encoded object with ``common_prefixes`` string field and ``entries`` list field.

.. code-block:: none

    $ curl http://localhost:8500/bzz-list:/679bde3ccb6fb911db96a0ea1586c04899c6c0cc6d3426e9ee361137b270a463/ | jq
    {
        "entries": [
            {
                "hash": "d0675100bc4580a0ad890b5d6f06310c0705d4ab1e796cfa1a8c597840f9793f",
                "path": "readme.md.1",
                "mode": 420,
                "size": 33,
                "mod_time": "2018-06-15T14:21:32+02:00"
            },
            {
                "hash": "f97cf36ac0dd7178c098f3661cd0402fcc711ff62b67df9893d29f1db35adac6",
                "path": "readme.md.2",
                "mode": 420,
                "size": 35,
                "mod_time": "2018-06-15T14:42:06+02:00"
            }
        ]
        }

bzz-hash
-------------

.. code-block:: none

    GET http://localhost:8500/bzz-hash:/theswarm.eth/

Swarm accepts GET requests for bzz-hash url scheme and responds with the hash value of the raw content, the same content returned by requests with bzz-raw scheme. Hash of the manifest is also the hash stored in ENS so bzz-hash can be used for ENS domain resolution.

Response content type is *text/plain*.

.. code-block:: none

    $ curl http://localhost:8500/bzz-hash:/theswarm.eth/
    7a90587bfc04ac4c64aeb1a96bc84f053d3d84cefc79012c9a07dd5230dc1fa4%

bzz-immutable
-------------

.. code-block:: none

    GET http://localhost:8500/bzz-immutable:/2477cc8584cc61091b5cc084cdcdb45bf3c6210c263b0143f030cf7d750e894d

The same as the generic scheme but there is no ENS domain resolution, the domain part of the path needs to be a valid hash. This is also a read-only scheme but explicit in its integrity protection. A particular bzz-immutable url will always necessarily address the exact same fixed immutable content.

.. code-block:: none

    $ curl http://localhost:8500/bzz-immutable:/679bde3ccb6fb911db96a0ea1586c04899c6c0cc6d3426e9ee361137b270a463/readme.md.1
    ## Hello Swarm!

    Swarm is awesome%
    $ curl -H "Accept:application/json" http://localhost:8500/bzz-immutable:/theswarm.eth/ | jq .
    {
        "Msg": "cannot resolve theswarm.eth: immutable address not a content hash: \"theswarm.eth\"",
        "Code": 404,
        "Timestamp": "Fri, 15 Jun 2018 13:22:27 UTC",
        "Details": ""
    }


