.. _contribute:

Contribute to Docs
==================

This documentation has been build using the Python `Sphinx <http://www.sphinx-doc.org/>`_
documentation tool.

Since the `Ethereum tests <https://github.com/ethereum/tests>`_ repository is very
large to clone locally, a convenient way to contribute to the documentation is to 
make a fork of the test repo, add the changes online with the GitHub 
`reStructuredText <http://www.sphinx-doc.org/en/stable/rest.html>`_ editor
and then open a PR.

If you want to clone to your desk you might want to make use of ``git clone --depth 1``
for faster download.

You can build the documentation by running ``make html`` from the ``docs`` directory
in the tests repository.
