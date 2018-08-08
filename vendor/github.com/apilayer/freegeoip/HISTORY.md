# History of freegeoip.net

The freegeoip software is the result of a web server research project that
started in 2009, written in Python and hosted on
[Google App Engine](http://appengine.google.com). It was rapidly adopted by
many developers around the world due to its simplistic and straightforward
HTTP API, causing the free account on GAE to exceed its quota every day
after few hours of operation.

A year later freegeoip 1.0 was released, and the freegeoip.net domain
moved over to its own server infrastructure. The software was rewritten
using the [Cyclone](http://cyclone.io) web framework, backed by
[Twisted](http://twistedmatrix.com) and [PyPy](http://pypy.org) in
production. That's when the first database management tool was created,
a script that would download many pieces of information from the Internet
to create the IP database, an sqlite flat file used by the server.

This version of the Python server shipped with a much better front-end as
well, but still as a server-side rendered template inherited from the GAE
version. It was only circa 2011 that freegeoip got its first standalone
front-end based on jQuery, and is when Twitter bootstrap was first used.

Python played an important role in the early life of freegeoip and
allowed the service to grow and evolve fast. It provided a lot of
flexibility in building and maintaining the IP database using multiple
sources of data. This version of the server lasted until 2013, when
it was once again rewritten from scratch, this time in Go. The database
tool, however, remained intact.

In 2013 the Go version was released as freegeoip 2.0 and this version
had many iterations. The first versions of the server written in Go were
very rustic, practically a verbatim transcription of the Python server.
Took a while until it started looking more like common Go code, and to
have tests.

Another important change that shipped with v2 was a front-end based on
AngularJS, but still mixed with some jQuery. The Google map in the front
page was made optional to put more focus on the HTTP API. The popularity
of freegeoip has increased considerably over the years of 2013 and 2014,
calling for more.

Enter freegeoip 3.0, an evolution of the Go server. The foundation of
freegeoip, which is the IP database and HTTP API, now lives in a Go
package that other developers can leverage. The freegeoip web server is
built on this package making its code cleaner, the server faster,
and requires zero maintenance for the IP database. The server downloads
the file from MaxMind and keep it up to date in background.

This and other changes make it very Docker friendly.

The front-end has been trimmed down to a single index.html file that loads
CSS and JS from CDNs on the internet. The JS part is based on AngularJS
and handles the search request and response of the public site. The
optional map has become a link to Google Maps following the lat/long
of the query results.
