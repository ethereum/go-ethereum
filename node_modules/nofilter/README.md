[![Tests](https://github.com/hildjj/nofilter/actions/workflows/node.js.yml/badge.svg)](https://github.com/hildjj/nofilter/actions/workflows/node.js.yml)
[![coverage](https://codecov.io/gh/hildjj/nofilter/branch/main/graph/badge.svg?token=7BdD02c03C)](https://codecov.io/gh/hildjj/nofilter)

# NoFilter

A node.js package to read and write a stream of data into or out of what looks
like a growable [Buffer](https://nodejs.org/api/buffer.html).

I kept needing this, and none of the existing packages seemed to have enough
features, test coverage, etc.

# Examples

As a data sink:
```js
const NoFilter = require('nofilter')
// In ES6:
// import NoFilter from 'nofilter'
// In typescript:
// import NoFilter = require('nofilter')

const nf = new NoFilter()
nf.on('finish', () => {
  console.log(nf.toString('base64'))
})
process.stdin.pipe(nf)
```

As a data source:
```js
const NoFilter = require('nofilter')
const nf = new NoFilter('010203', 'hex')
nf.pipe(process.stdout)
```

Read the [API Docs](http://hildjj.github.io/nofilter/).
