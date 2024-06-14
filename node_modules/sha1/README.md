sha1
====

[![build
status](https://secure.travis-ci.org/pvorb/node-sha1.png)](http://travis-ci.org/pvorb/node-sha1)

a native js function for hashing messages with the SHA-1 algorithm


Installation
------------

    npm install sha1

or

    ender build sha1


Example
-------

~~~ javascript
var sha1 = require('sha1');

sha1("message");
~~~

This will return the string:

    "6f9b9af3cd6e8b8a73c2cdced37fe9f59226e27d"


API
---

### sha1(msg)

Returns the SHA-1 hash of the given message.

  * `msg` String -- the message that you want to hash.

It's as simple as that.


Credits
-------

This function is taken from [CryptoJS](http://code.google.com/p/crypto-js/).
This package only provides easy access to a javascript-only version of the SHA-1
algorithm over npm.


License
-------

(New BSD License /
[BSD 3-Clause License](http://opensource.org/licenses/BSD-3-Clause))

~~~
Copyright © 2009, Jeff Mott. All rights reserved.
Copyright © 2011, Paul Vorbach. All rights reserved.

All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.
* Redistributions in binary form must reproduce the above copyright notice, this
  list of conditions and the following disclaimer in the documentation and/or
  other materials provided with the distribution.
* Neither the name Crypto-JS nor the names of its contributors may be used to
  endorse or promote products derived from this software without specific prior
  written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
~~~
