# req-cwd [![Build Status](https://travis-ci.org/sindresorhus/req-cwd.svg?branch=master)](https://travis-ci.org/sindresorhus/req-cwd)

> Require a module like [`require()`](https://nodejs.org/api/globals.html#globals_require) but from the current working directory


## Install

```
$ npm install --save req-cwd
```


## Usage

```js
const reqCwd = require('req-cwd');

// Target module is at '/Users/sindresorhus/unicorn/foo.js'

console.log(__dirname);
//=> '/Users/sindresorhus/rainbow'

console.log(process.cwd());
//=> '/Users/sindresorhus/unicorn'

const foo = reqCwd('./foo');
```


## API

### reqCwd(moduleId)

Like `require()`, throws when the module can't be found.

### reqCwd.silent(moduleId)

Returns `null` instead of throwing when the module can't be found.

#### moduleId

Type: `string`

What you would use in `require()`.


## Related

- [req-from](https://github.com/sindresorhus/req-from) - Require a module from a given path
- [resolve-from](https://github.com/sindresorhus/resolve-from) - Resolve the path of a module from a given path
- [resolve-cwd](https://github.com/sindresorhus/resolve-cwd) - Resolve the path of a module from the current working directory
- [resolve-pkg](https://github.com/sindresorhus/resolve-pkg) - Resolve the path of a package regardless of it having an entry point
- [lazy-req](https://github.com/sindresorhus/lazy-req) - Require modules lazily


## License

MIT © [Sindre Sorhus](https://sindresorhus.com)
