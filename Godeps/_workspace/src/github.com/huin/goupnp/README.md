goupnp is a UPnP client library for Go

Installation
------------

Run `go get -u github.com/huin/goupnp`.

Regenerating dcps generated source code:
----------------------------------------

1. Install gotasks: `go get -u github.com/jingweno/gotask`
2. Change to the gotasks directory: `cd gotasks`
3. Download UPnP specification data (if not done already): `wget http://upnp.org/resources/upnpresources.zip`
4. Regenerate source code: `gotask specgen -s upnpresources.zip -o ../dcps`
