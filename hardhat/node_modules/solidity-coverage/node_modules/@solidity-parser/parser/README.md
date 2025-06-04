# Solidity Parser for JavaScript

[![npm version](https://badge.fury.io/js/%40solidity-parser%2Fparser.svg)](https://badge.fury.io/js/%40solidity-parser%2Fparser)

A JavaScript package for parsing [Solidity](https://solidity.readthedocs.io/) code using [ANTLR (ANother Tool for Language Recognition)](https://www.antlr.org/) grammar.

This is a fork of [@federicobond](https://github.com/federicobond)'s original [repo](https://github.com/federicobond/solidity-parser-antlr),
with some extra features taken from [Consensys Diligence's alternative fork](https://github.com/consensys/solidity-parser-antlr).

## Installation

The following installation options assume [Node.js](https://nodejs.org/en/download/) has already been installed.

Using [Node Package Manager (npm)](https://www.npmjs.com/).

```
npm install @solidity-parser/parser
```

Using [yarn](https://yarnpkg.com/)

```
yarn add @solidity-parser/parser
```

## Usage

```javascript
const parser = require('@solidity-parser/parser')

const input = `
    contract test {
        uint256 a;
        function f() {}
    }
`
try {
  const ast = parser.parse(input)
  console.log(ast)
} catch (e) {
  if (e instanceof parser.ParserError) {
    console.error(e.errors)
  }
}
```

The `parse` method also accepts a second argument which lets you specify the
following options, in a style similar to the _esprima_ API:

| Key      | Type    | Default | Description                                                                                                                                                                                          |
| -------- | ------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| tolerant | Boolean | false   | When set to `true` it will collect syntax errors and place them in a list under the key `errors` inside the root node of the returned AST. Otherwise, it will raise a `parser.ParserError`.          |
| loc      | Boolean | false   | When set to `true`, it will add location information to each node, with start and stop keys that contain the corresponding line and column numbers. Column numbers start from 0, lines start from 1. |
| range    | Boolean | false   | When set to `true`, it will add range information to each node, which consists of a two-element array with start and stop character indexes in the input.                                            |

### Example with location information

```javascript
parser.parse('contract test { uint a; }', { loc: true })

// { type: 'SourceUnit',
//   children:
//    [ { type: 'ContractDefinition',
//        name: 'test',
//        baseContracts: [],
//        subNodes: [Array],
//        kind: 'contract',
//        loc: [Object] } ],
//   loc: { start: { line: 1, column: 0 }, end: { line: 1, column: 24 } } }
```

### Example using a visitor to walk over the AST

```javascript
var ast = parser.parse('contract test { uint a; }')

// output the path of each import found
parser.visit(ast, {
  ImportDirective: function (node) {
    console.log(node.path)
  },
})
```

## Usage in the browser

A browser-friendly version is available in `dist/index.umd.js` (along with its sourcemaps file) in the published version.

If you are using webpack, keep in mind that minimizing your bundle will mangle function names, breaking the parser. To fix this you can just set `optimization.minimize` to `false`.

## Contribution

This project is dependant on the [@solidity-parser/antlr](https://github.com/solidity-parser/antlr) repository via a git submodule. To clone this repository and the submodule, run

```
git clone --recursive
```

If you have already cloned this repo, you can load the submodule with

```
git submodule update --init
```

This project can be linked to a forked `@solidity-parser/antlr` project by editing the url in the [.gitmodules](.gitmodules) file to point to the forked repo and running

```
git submodule sync
```

The Solidity ANTLR file [Solidity.g4](./antlr/Solidity.g4) can be built with the following. This will also download the ANTLR Java Archive (jar) file to `antlr/antlr4.jar` if it doesn't already exist. The generated ANTLR tokens and JavaScript files are copied the [src](./src) folder.

```
yarn run antlr
```

The files to be distributed with the npm package are in the `dist` folder and built by running

```
yarn run build
```

The [mocha](https://mochajs.org/) tests under the [test](./test) folder can be run with the following. This includes parsing the [test.sol](./test/test.sol) Solidity file.

```
yarn run test
```

## Used by

- [Hardhat](https://hardhat.org/)
- [sol2uml](https://github.com/naddison36/sol2uml)
- [Solhint](https://github.com/protofire/solhint/)
- [solidity-coverage](https://github.com/sc-forks/solidity-coverage)
- [prettier-solidity](https://github.com/prettier-solidity/prettier-plugin-solidity/)
- [eth-gas-reporter](https://github.com/cgewecke/eth-gas-reporter)

## License

[MIT](./LICENSE)
