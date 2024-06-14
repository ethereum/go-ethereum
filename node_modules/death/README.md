Node.js - death
================

Gracefully cleanup when termination signals are sent to your process.


Why?
----

Because adding clean up callbacks for `uncaughtException`, `SIGINT`, and `SIGTERM` is annoying. Ideally, you can
use this package to put your cleanup code in one place and exit gracefully if you need to.


Operating System Compatibility
------------------------------

It's only been tested on POSIX compatible systems. [Here's a nice discussion](https://github.com/joyent/node/issues/1553) on Windows signals, apparently, this has been fixed/mapped. 


Installation
------------

    npm install death



Example
------

```js
var ON_DEATH = require('death'); //this is intentionally ugly

ON_DEATH(function(signal, err) {
  //clean up code here
})
```


Usage
-----

By default, it sets the callback on `SIGINT`, `SIGQUIT`, and `SIGTERM`.

### Signals
- **SIGINT**: Sent from CTRL-C
- **SIGQUIT**: Sent from keyboard quit action.
- **SIGTERM**: Sent from operating system `kill`.

More discussion and detail: http://www.gnu.org/software/libc/manual/html_node/Termination-Signals.html and http://pubs.opengroup.org/onlinepubs/009695399/basedefs/signal.h.html and http://pubs.opengroup.org/onlinepubs/009695399/basedefs/xbd_chap11.html.

AS they pertain to Node.js: http://dailyjs.com/2012/03/15/unix-node-signals/


#### Want to catch uncaughtException?

No problem, do this:

```js
var ON_DEATH = require('death')({uncaughtException: true}) 
```

#### Want to know which signals are being caught?

Do this:

```js
var ON_DEATH = require('death')({debug: true})
```

Your process will then log anytime it catches these signals.

#### Want to catch SIGHUP?

Be careful with this one though. Typically this is fired if your SSH connection dies, but can
also be fired if the program is made a daemon. 

Do this:

```js
var ON_DEATH = require('death')({SIGHUP: true})
```

#### Why choose the ugly "ON_DEATH"?

Name it whatever you want. I like `ON_DEATH` because it stands out like a sore thumb in my code.


#### Want to remove event handlers?

If you want to remove event handlers `ON_DEATH` returns a function for cleaning
up after itself:

```js
var ON_DEATH = require('death')
var OFF_DEATH = ON_DEATH(function(signal, err) {
  //clean up code here
})

// later on...
OFF_DEATH();
```

License
-------

(MIT License)

Copyright 2012, JP Richardson  <jprichardson@gmail.com>


