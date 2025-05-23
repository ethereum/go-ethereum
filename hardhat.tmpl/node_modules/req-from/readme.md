# req-from [![Build Status](https://travis-ci.org/sindresorhus/req-from.svg?branch=master)](https://travis-ci.org/sindresorhus/req-from)

> Require a module like [`require()`](https://nodejs.org/api/globals.html#globals_require) but from a given path


## Install

```
$ npm install --save req-from
```


## Usage

```js
const reqFrom = require('req-from');

// There is a file at `./foo/bar.js`

reqFrom('foo', './bar');
```


## API

### reqFrom(fromDir, moduleId)

Like `require()`, throws when the module can't be found.

### reqFrom.silent(fromDir, moduleId)

Returns `null` instead of throwing when the module can't be found.

#### fromDir

Type: `string`

Directory to require from.

#### moduleId

Type: `string`

What you would use in `require()`.


## Tip

Create a partial using a bound function if you want to require from the same `fromDir` multiple times:

```js
const reqFromFoo = reqFrom.bind(null, 'foo');

reqFromFoo('./bar');
reqFromFoo('./baz');
```


## Related

- [req-cwd](https://github.com/sindresorhus/req-cwd) - Require a module from the current working directory
- [resolve-from](https://github.com/sindresorhus/resolve-from) - Resolve the path of a module from a given path
- [resolve-cwd](https://github.com/sindresorhus/resolve-cwd) - Resolve the path of a module from the current working directory
- [resolve-pkg](https://github.com/sindresorhus/resolve-pkg) - Resolve the path of a package regardless of it having an entry point
- [lazy-req](https://github.com/sindresorhus/lazy-req) - Require modules lazily


## License

MIT © [Sindre Sorhus](https://sindresorhus.com)
