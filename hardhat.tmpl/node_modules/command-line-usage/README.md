[![view on npm](https://badgen.net/npm/v/command-line-usage)](https://www.npmjs.org/package/command-line-usage)
[![npm module downloads](https://badgen.net/npm/dt/command-line-usage)](https://www.npmjs.org/package/command-line-usage)
[![Gihub repo dependents](https://badgen.net/github/dependents-repo/75lb/command-line-usage)](https://github.com/75lb/command-line-usage/network/dependents?dependent_type=REPOSITORY)
[![Gihub package dependents](https://badgen.net/github/dependents-pkg/75lb/command-line-usage)](https://github.com/75lb/command-line-usage/network/dependents?dependent_type=PACKAGE)
[![Node.js CI](https://github.com/75lb/command-line-usage/actions/workflows/node.js.yml/badge.svg)](https://github.com/75lb/command-line-usage/actions/workflows/node.js.yml)
[![Coverage Status](https://coveralls.io/repos/github/75lb/command-line-usage/badge.svg)](https://coveralls.io/github/75lb/command-line-usage)
[![js-standard-style](https://img.shields.io/badge/code%20style-standard-brightgreen.svg)](https://github.com/feross/standard)

# command-line-usage

***Upgraders, please check the [release notes](https://github.com/75lb/command-line-usage/releases).***

A simple, data-driven module for creating a usage guide.

## Synopsis

A usage guide is created by first defining an arbitrary number of sections, e.g. a description section, synopsis, option list, examples, footer etc. Each section has an optional header, some content and must be of type [`content`](https://github.com/75lb/command-line-usage/blob/master/doc/api.md#module_command-line-usage--commandLineUsage..content) or [`optionList`](https://github.com/75lb/command-line-usage/blob/master/doc/api.md#module_command-line-usage--commandLineUsage..optionList). This section data is passed to [`commandLineUsage()`](https://github.com/75lb/command-line-usage/blob/master/doc/api.md#exp_module_command-line-usage--commandLineUsage) which returns a usage guide.

Inline ansi formatting can be used anywhere within section content using [chalk template literal syntax](https://github.com/chalk/chalk/tree/v2.4.2#tagged-template-literal).

For example, this script:
```js
import commandLineUsage from 'command-line-usage'

const sections = [
  {
    header: 'A typical app',
    content: 'Generates something {italic very} important.'
  },
  {
    header: 'Options',
    optionList: [
      {
        name: 'input',
        typeLabel: '{underline file}',
        description: 'The input to process.'
      },
      {
        name: 'help',
        description: 'Print this usage guide.'
      }
    ]
  }
]
const usage = commandLineUsage(sections)
console.log(usage)
```

Outputs this guide:

<img src="https://raw.githubusercontent.com/75lb/command-line-usage/master/example/screens/synopsis.png" width="90%">

## Some examples

### Typical

A fairly typical usage guide with three sections - description, option list and footer. [Code](https://github.com/75lb/command-line-usage/wiki/How-to-create-a-typical-usage-guide).

<img src="https://raw.githubusercontent.com/75lb/command-line-usage/master/example/screens/simple.png" width="90%">

### Option List groups

Demonstrates breaking the option list up into groups. [Code](https://github.com/75lb/command-line-usage/wiki/How-to-break-the-option-list-up-into-groups).

<img src="https://raw.githubusercontent.com/75lb/command-line-usage/master/example/screens/groups.png" width="90%">

### Banners

A banner is created by adding the `raw: true` property to your `content`. This flag disables any formatting on the content, displaying it raw as supplied.

#### Header

Demonstrates a banner at the top. This example also adds a `synopsis` section. [Code](https://github.com/75lb/command-line-usage/wiki/How-to-add-a-banner-to-your-usage-guide#code).

<img src="https://raw.githubusercontent.com/75lb/command-line-usage/master/example/screens/header.png" width="90%">

#### Footer

Demonstrates a footer banner. [Code](https://github.com/75lb/command-line-usage/wiki/How-to-add-a-banner-to-your-usage-guide#code-1).

<img src="https://raw.githubusercontent.com/75lb/command-line-usage/master/example/screens/footer.png" width="90%">

### Examples section (table layout)

An examples section is added. To achieve this table layout, supply the `content` as an array of objects. The property names of each object are not important, so long as they are consistent throughout the array. [Code](https://github.com/75lb/command-line-usage/wiki/How-to-add-an-examples-section-to-your-usage-guide).

<img src="https://raw.githubusercontent.com/75lb/command-line-usage/master/example/screens/example-columns.png" width="90%">

### Advanced optionList layout

The `optionList` layout is fully configurable by setting the `tableOptions` property with an options object suitable for passing into [table-layout](https://github.com/75lb/table-layout#table-). This example overrides the default column widths and adds flame padding. [Code](https://github.com/75lb/command-line-usage/wiki/How-to-use-advanced-optionList-table-formatting).

<img src="https://raw.githubusercontent.com/75lb/command-line-usage/master/example/screens/option-list-options.png" width="90%">

### Command list

Useful if your app is command-driven, like git or npm. [Code](https://github.com/75lb/command-line-usage/wiki/How-to-add-a-command-list-to-your-usage-guide).

<img src="https://raw.githubusercontent.com/75lb/command-line-usage/master/example/screens/command-list.png" width="90%">

### Description section (table layout)

Demonstrates supplying specific [table layout](https://github.com/75lb/table-layout) options to achieve more advanced layout. In this case the second column (containing the hammer and sickle) has a fixed `width` of 40 and `noWrap` enabled (as the input is already formatted as desired). [Code](https://github.com/75lb/command-line-usage/wiki/How-to-add-a-description-section-to-your-usage-guide).

<img src="https://raw.githubusercontent.com/75lb/command-line-usage/master/example/screens/description-columns.png" width="90%">

### Real-life

The [polymer-cli](https://github.com/Polymer/tools/tree/master/packages/cli) usage guide is a good real-life example.

<img src="https://raw.githubusercontent.com/75lb/command-line-usage/master/example/screens/polymer.png" width="90%">

## Documentation

* [API Reference](https://github.com/75lb/command-line-usage/blob/master/doc/api.md)
* [The full list of examples](https://github.com/75lb/command-line-usage/wiki)

* * *

&copy; 2015-22 Lloyd Brookes \<75pound@gmail.com\>. Documented by [jsdoc-to-markdown](https://github.com/75lb/jsdoc-to-markdown).
