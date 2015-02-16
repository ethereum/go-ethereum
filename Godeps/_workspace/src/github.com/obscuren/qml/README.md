# QML support for the Go language

Documentation
-------------

The introductory documentation as well as the detailed API documentation is
available at [gopkg.in/qml.v1](http://godoc.org/gopkg.in/qml.v1).


Blog posts
----------

Some relevant blog posts:

  * [Announcing qml v1 for Go](http://blog.labix.org/2014/08/13/announcing-qml-v1-for-go)
  * [Packing resources into Go qml binaries](http://blog.labix.org/2014/09/26/packing-resources-into-go-qml-binaries)
  * [Go qml contest results](http://blog.labix.org/2014/04/25/qml-contest-results)
  * [Arbitrary Qt extensions with Go qml](http://blog.labix.org/2014/03/21/arbitrary-qt-extensions-with-go-qml)
  * [The new Go qml OpenGL API](http://blog.labix.org/2014/08/29/the-new-go-qml-opengl-api)
  * [QML components with Go and OpenGL](http://blog.labix.org/2013/12/23/qml-components-with-go-and-opengl)


Videos
------

These introductory videos demonstrate the use of Go QML:

  * [Initial demo and overview](http://youtu.be/FVQlMrPa7lI)
  * [Initial demo running on an Ubuntu Touch phone](http://youtu.be/HB-3o8Cysec)
  * [Spinning Gopher with Go + QML + OpenGL](http://youtu.be/qkH7_dtOyPk)
  * [SameGame QML tutorial in Go](http://youtu.be/z8noX48hiMI)


Community
---------

Please join the [mailing list](https://groups.google.com/forum/#!forum/go-qml) for
following relevant development news and discussing project details.


Installation
------------

To try the alpha release you'll need:

  * Go >= 1.2, for the C++ support of _go build_
  * Qt 5.0.X or 5.1.X with the development files
  * The Qt headers qmetaobject_p.h and qmetaobjectbuilder_p.h, for the dynamic meta object support

See below for more details about getting these requirements installed in different environments and operating systems.

After the requirements are satisfied, _go get_ should work as usual:

    go get gopkg.in/qml.v1


Requirements on Ubuntu
----------------------

If you are using Ubuntu, the [Ubuntu SDK](http://developer.ubuntu.com/get-started/) will take care of the Qt dependencies:

    $ sudo add-apt-repository ppa:ubuntu-sdk-team/ppa
    $ sudo apt-get update
    $ sudo apt-get install qtdeclarative5-dev qtbase5-private-dev qtdeclarative5-private-dev libqt5opengl5-dev qtdeclarative5-qtquick2-plugin

and Go >= 1.2 may be installed using [godeb](http://blog.labix.org/2013/06/15/in-flight-deb-packages-of-go):

    $ # Pick the right one for your system: 386 or amd64
    $ ARCH=amd64
    $ wget -q https://godeb.s3.amazonaws.com/godeb-$ARCH.tar.gz
    $ tar xzvf godeb-$ARCH.tar.gz
    godeb
    $ sudo mv godeb /usr/local/bin
    $ godeb install
    $ go get gopkg.in/qml.v1


Requirements on Ubuntu Touch
----------------------------

After following the [installation instructions](https://wiki.ubuntu.com/Touch/Install) for Ubuntu Touch,
run the following commands to get a working build environment inside the device:

    $ adb shell
    # cd /tmp
    # wget https://github.com/go-qml/qml/raw/v1/cmd/ubuntu-touch/setup.sh
    # /bin/bash setup.sh
    # su - phablet
    $

At the end of setup.sh, the phablet user will have GOPATH=$HOME in the environment,
the qml package will be built, and the particle example will be built and run. For
stopping it from the command line, run as the phablet user:

    $ ubuntu-app-stop gopkg.in.qml.particle-example

for running it again:

    $ ubuntu-app-launch gopkg.in.qml.particle-example

These commands depend on the following file, installed by setup.sh:

    ~/.local/share/applications/gopkg.in.qml.particle-example.desktop


Requirements on Mac OS X
------------------------

On Mac OS X you'll need QT5. It's easiest to install with Homebrew, a
third-party package management system for OS X.

Installation instructions for Homebrew are here:

    http://brew.sh/

Then, install the qt5 and pkg-config packages:

    $ brew install qt5 pkg-config

Then, force brew to "link" qt5 (this makes it available under /usr/local):

    $ brew link --force qt5

And finally, fetch and install go-qml:

    $ go get gopkg.in/qml.v1


Requirements on Windows
-----------------------

On Windows you'll need the following:

  * [MinGW gcc](http://sourceforge.net/projects/mingw/files/latest/download) 4.8.1 (install mingw-get and install the gcc from within the setup GUI)
  * [Qt 5.1.1](http://download.qt-project.org/official_releases/qt/5.1/5.1.1/qt-windows-opensource-5.1.1-mingw48_opengl-x86-offline.exe) for MinGW 4.8
  * [Go >= 1.2](http://golang.org/doc/install)

Then, assuming Qt was installed under `C:\Qt5.1.1\`, set up the following environment variables in the respective configuration:

    CPATH += C:\Qt5.1.1\5.1.1\mingw48_32\include
    LIBRARY_PATH += C:\Qt5.1.1\5.1.1\mingw48_32\lib
    PATH += C:\Qt5.1.1\5.1.1\mingw48_32\bin

After reopening the shell for the environment changes to take effect, this should work:

    go get gopkg.in/qml.v1


Requirements everywhere else
----------------------------

If your operating system does not offer these dependencies readily,
you may still have success installing [Go >= 1.2](http://golang.org/doc/install)
and [Qt 5.0.2](http://download.qt-project.org/archive/qt/5.0/5.0.2/)
directly from the upstreams.  Note that you'll likely have to adapt
environment variables to reflect the custom installation path for
these libraries. See the instructions above for examples.
