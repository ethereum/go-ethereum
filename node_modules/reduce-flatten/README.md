[![view on npm](http://img.shields.io/npm/v/reduce-flatten.svg)](https://www.npmjs.org/package/reduce-flatten)
[![npm module downloads](http://img.shields.io/npm/dt/reduce-flatten.svg)](https://www.npmjs.org/package/reduce-flatten)
[![Build Status](https://travis-ci.org/75lb/reduce-flatten.svg?branch=master)](https://travis-ci.org/75lb/reduce-flatten)
[![Dependency Status](https://david-dm.org/75lb/reduce-flatten.svg)](https://david-dm.org/75lb/reduce-flatten)
[![js-standard-style](https://img.shields.io/badge/code%20style-standard-brightgreen.svg)](https://github.com/feross/standard)

<a name="module_reduce-flatten"></a>

## reduce-flatten
Flatten an array into the supplied array.

**Example**  
```js
const flatten = require('reduce-flatten')
```
<a name="exp_module_reduce-flatten--flatten"></a>

### flatten() ⏏
**Kind**: Exported function  
**Example**  
```js
> numbers = [ 1, 2, [ 3, 4 ], 5 ]
> numbers.reduce(flatten, [])
[ 1, 2, 3, 4, 5 ]
```

* * *

&copy; 2016-18 Lloyd Brookes \<75pound@gmail.com\>. Documented by [jsdoc-to-markdown](https://github.com/jsdoc2md/jsdoc-to-markdown).
