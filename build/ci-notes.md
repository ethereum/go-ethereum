# Debian Packaging

Tagged releases and develop branch commits are available as installable Debian packages
for Ubuntu. Packages are built for the all Ubuntu versions which are supported by
Canonical.

Packages of develop branch commits have suffix -unstable and cannot be installed alongside
the stable version. Switching between release streams requires user intervention.

## Launchpad

The packages are built and served by launchpad.net. We generate a Debian source package
for each distribution and upload it. Their builder picks up the source package, builds it
and installs the new version into the PPA repository. Launchpad requires a valid signature
by a team member for source package uploads.

The signing key is stored in an environment variable which Travis CI makes available to
certain builds. Since Travis CI doesn't support FTP, SFTP is used to transfer the
packages. To set this up yourself, you need to create a Launchpad user and add a GPG key
and SSH key to it. Then encode both keys as base64 and configure 'secret' environment
variables `PPA_SIGNING_KEY` and `PPA_SSH_KEY` on Travis.

We want to build go-ethereum with the most recent version of Go, irrespective of the Go
version that is available in the main Ubuntu repository. In order to make this possible,
our we depend on ~gophers/ubuntu/archive and ~longsleep/ubuntu/golang-backports PPAs. Our
source package build-depends on golang-1.11 on Tusty and golang-1.13 on others, which is
co-installable alongside the regular golang package. PPA dependencies can be edited at
https://launchpad.net/%7Eethereum/+archive/ubuntu/ethereum/+edit-dependencies

## Building Packages Locally (for testing)

You need to run Ubuntu to do test packaging.

Add the longsleep PPA and install Go 1.13 and Debian packaging tools:

    $ sudo apt-add-repository ppa:longsleep/ubuntu/golang-backports
    $ sudo apt-get update
    $ sudo apt-get install build-essential golang-1.13 devscripts debhelper python-bzrlib python-paramiko

Create the source packages:

    $ go run build/ci.go debsrc -workdir dist

Then go into the source package directory for your running distribution and build the package:

    $ cd dist/ethereum-unstable-1.9.6+bionic
    $ dpkg-buildpackage

Built packages are placed in the dist/ directory.

    $ cd ..
    $ dpkg-deb -c geth-unstable_1.9.6+bionic_amd64.deb
