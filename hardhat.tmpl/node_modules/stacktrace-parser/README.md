[![Build Status][build-image]][build-url] [![dependencies][deps-image]][deps-url] [![dev-dependencies][dev-deps-image]][dev-deps-url]

# StackTrace-Parser

This parser parses a stack trace from any browser or Node.js and returns an array of hashes each representing a line.

The goal here is to support every browser even old Internet Explorer stack traces will work.

## Install

```bashv0.1
npm install stacktrace-parser
```

## Usage

```JavaScript
import * as stackTraceParser from 'stacktrace-parser';

try {
  throw new Error('My error');
} catch(ex) {
  const stack = stackTraceParser.parse(ex.stack);
}
```

Every line contains five properties: `lineNumber`, `methodName`, `arguments`, `file` and `column` (if applicable).

## TODOs

- parse stack traces from other sources (Ruby, etc) (v0.3)

## Contribution

If you want to contrib, then do you thing, write tests, run `npm run test` ensure that everything is green,
commit and make the pull request. Or just write an issue, or let's talk.

## Contributors

- [Georg Tavonius](https://github.com/calamari)
- [James Ide](https://github.com/ide)
- [Alexander Kotliarskyi](https://github.com/frantic)
- [Dimitri Benin](https://github.com/BendingBender)
- [Tony Brix](https://github.com/UziTech)

## LICENSE

[The MIT License (MIT)](https://github.com/errwischt/stacktrace-parser/blob/master/LICENSE)

[build-image]: https://img.shields.io/travis/errwischt/stacktrace-parser/master.svg?style=flat-square
[build-url]: https://travis-ci.org/errwischt/stacktrace-parser
[deps-image]: https://img.shields.io/david/errwischt/stacktrace-parser.svg?style=flat-square
[deps-url]: https://david-dm.org/errwischt/stacktrace-parser
[dev-deps-image]: https://img.shields.io/david/dev/errwischt/stacktrace-parser.svg?style=flat-square
[dev-deps-url]: https://david-dm.org/errwischt/stacktrace-parser?type=dev
