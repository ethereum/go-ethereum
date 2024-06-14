[![npm version](https://badge.fury.io/js/ethereum-bloom-filters.svg)](https://badge.fury.io/js/ethereum-bloom-filters)
![downloads](https://img.shields.io/npm/dw/ethereum-bloom-filters)

# ethereum-bloom-filters

A lightweight bloom filter client which allows you to test ethereum blooms for fast checks of set membership.

This package only has 1 dependency which is on `@noble/hashes` which has no dependencies on at all and is funded by ethereum foundation.

## Installation

### npm:

```js
$ npm install ethereum-bloom-filters
```

### yarn:

```js
$ yarn add ethereum-bloom-filters
```

## Usage

### JavaScript (ES3)

```js
var ethereumBloomFilters = require('ethereum-bloom-filters');
```

### JavaScript (ES5 or ES6)

```js
const ethereumBloomFilters = require('ethereum-bloom-filters');
```

### JavaScript (ES6) / TypeScript

```js
import {
  isBloom,
  isUserEthereumAddressInBloom,
  isContractAddressInBloom,
  isTopic,
  isTopicInBloom,
  isInBloom,
} from 'ethereum-bloom-filters';
```

### Including within a web application which doesn't use any transpiler

When using angular, react or vuejs these frameworks handle dependencies and transpile them so they work on the web, so if you're using any of them just use the above code snippets to start using this package.

If you're using a standard web application you can go [here](https://github.com/joshstevens19/ethereum-bloom-filters/tree/master/web-scripts) to copy any of the versioned script files and then dropping it into your web application, making sure you reference it within a script tag in the head of the website.

This will expose the library as a global variable named `ethereumBloomFilters`, you can then execute the methods through this variable:

```js
ethereumBloomFilters.isBloom(...)
ethereumBloomFilters.isUserEthereumAddressInBloom(...)
ethereumBloomFilters.isContractAddressInBloom(...)
ethereumBloomFilters.isTopic(...)
ethereumBloomFilters.isTopicInBloom(...)
ethereumBloomFilters.isInBloom(...)
```

You can find out more about the functions parameters below.

We do not expose an cdn for security reasons.

## What are bloom filters?

A Bloom filter is a probabilistic, space-efficient data structure used for fast checks of set membership. That probably doesn’t mean much to you yet, and so let’s explore how bloom filters might be used.

Imagine that we have some large set of data, and we want to be able to quickly test if some element is currently in that set. The naive way of checking might be to query the set to see if our element is in there. That’s probably fine if our data set is relatively small. Unfortunately, if our data set is really big, this search might take a while. Luckily, we have tricks to speed things up in the ethereum world!

A bloom filter is one of these tricks. The basic idea behind the Bloom filter is to hash each new element that goes into the data set, take certain bits from this hash, and then use those bits to fill in parts of a fixed-size bit array (e.g. set certain bits to 1). This bit array is called a bloom filter.

Later, when we want to check if an element is in the set, we simply hash the element and check that the right bits are in the bloom filter. If at least one of the bits is 0, then the element definitely isn’t in our data set! If all of the bits are 1, then the element might be in the data set, but we need to actually query the database to be sure. So we might have false positives, but we’ll never have false negatives. This can greatly reduce the number of database queries we have to make.

## ethereum-bloom-filters benefits with an real life example

A ethereum real life example in where this is useful is if you want to update a users balance on every new block so it stays as close to real time as possible. Without using a bloom filter on every new block you would have to force the balances even if that user may not of had any activity within that block. But if you use the logBlooms from the block you can test the bloom filter against the users ethereum address before you do any more slow operations, this will dramatically decrease the amount of calls you do as you will only be doing those extra operations if that ethereum address is within that block (minus the false positives outcome which will be negligible). This will be highly performant for your app.

## Requirements for blooms to be queryable

Blooms do not work with eth transactions (purely sending eth), eth transactions do not emit logs so do not exist in the bloom filter. This is what ethereum did purposely but it means you should query the eth balance every block to make sure it's in sync. Blooms will only work if the transaction emits an event which then ends up in the logs. The bloom filter is there to help you find logs. A contract can be written which does not emit an event and in that case, would not be queryable from a bloom filter. The erc20 token spec requires you to fire an event on `approval` and `transfer` so blooms will work for `approval` and `transfer` for ALL erc20 tokens, this will be most people's primary use-case. Saying that this can be used in any way you want with any use-case as long as events are emitted then it's queryable.

## Functions

### isBloom

```ts
isBloom(bloom: string): boolean;
```

Returns true if the bloom is a valid bloom.

### isUserEthereumAddressInBloom

```ts
isUserEthereumAddressInBloom(bloom: string, ethereumAddress: string): boolean;
```

Returns true if the ethereum users address is part of the given bloom
note: false positives are possible.

### isContractAddressInBloom

```ts
isContractAddressInBloom(bloom: string, contractAddress: string): boolean;
```

Returns true if the contract address is part of the given bloom
note: false positives are possible.

### isTopic

```ts
isTopic(topic: string): boolean;
```

Returns true if the topic is valid

### isTopicInBloom

```ts
isTopicInBloom(bloom: string, topic: string): boolean;
```

Returns true if the topic is part of the given bloom
note: false positives are possible.

### isInBloom

This is the raw base method which the other bloom methods above use. You can pass in a bloom and a value which will return true if its part of the given bloom.

```ts
isInBloom(bloom: string, value: string | Uint8Array): boolean;
```

Returns true if the value is part of the given bloom
note: false positives are possible.

## Issues

Please raise any issues in the below link.

https://github.com/joshstevens19/ethereum-bloom-filters/issues

## Thanks And Support

This package is brought to you by [Josh Stevens](https://github.com/joshstevens19). My aim is to be able to keep creating these awesome packages to help the Ethereum space grow with easier-to-use tools to allow the learning curve to get involved with blockchain development easier and making Ethereum ecosystem better. If you want to help with that vision and allow me to invest more time into creating cool packages or if this package has saved you a lot of development time donations are welcome, every little helps. By donating, you are supporting me to be able to maintain existing packages, extend existing packages (as Ethereum matures), and allowing me to build more packages for Ethereum due to being able to invest more time into it. Thanks, everyone!

## Direct donations

Direct donations any token accepted - Eth address > `0x699c2daD091ffcF18f3cd9E8495929CA3a64dFe1`

## Github sponsors

[sponsor me](https://github.com/sponsors/joshstevens19) via github using fiat money

## Contributors dev guide

To run locally firstly run:

```js
$ npm install
```

To build:

```js
$ tsc
```

To watch build:

```js
$ tsc --watch
```

To run tests:

```js
$ npm test
```
