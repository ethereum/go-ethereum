The Ethers Project
==================

[![npm (tag)](https://img.shields.io/npm/v/ethers)](https://www.npmjs.com/package/ethers)
[![Node.js CI](https://github.com/ethers-io/ethers.js/workflows/Node.js%20CI/badge.svg?branch=ethers-v5-beta)](https://github.com/ethers-io/ethers.js/actions?query=workflow%3A%22Node.js+CI%22)

A complete Ethereum wallet implementation and utilities in JavaScript (and TypeScript).

**Features:**

- Keep your private keys in your client, **safe** and sound
- Import and export **JSON wallets** (Geth, Parity and crowdsale)
- Import and export BIP 39 **mnemonic phrases** (12 word backup phrases) and **HD Wallets** (English as well as Czech, French, Italian, Japanese, Korean, Simplified Chinese, Spanish, Traditional Chinese)
- Meta-classes create JavaScript objects from any contract ABI, including **ABIv2** and **Human-Readable ABI**
- Connect to Ethereum nodes over [JSON-RPC](https://github.com/ethereum/wiki/wiki/JSON-RPC), [INFURA](https://infura.io), [Etherscan](https://etherscan.io), [Alchemy](https://alchemyapi.io) or [MetaMask](https://metamask.io)
- **ENS names** are first-class citizens; they can be used anywhere an Ethereum addresses can be used
- **Tiny** (~104kb compressed; 322kb uncompressed)
- **Modular** packages; include only what you need
- **Complete** functionality for all your Ethereum desires
- Extensive [documentation](https://docs.ethers.io/v5/)
- Large collection of **test cases** which are maintained and added to
- Fully **TypeScript** ready, with definition files and full TypeScript source
- **MIT License** (including ALL dependencies); completely open source to do with as you please


Keep Updated
------------

For the latest news and advisories, please follow the [@ethersproject](https://twitter.com/ethersproject)
on Twitter (low-traffic, non-marketing, important information only) as well as watch this GitHub project.

For the latest changes, see the [CHANGELOG](https://github.com/ethers-io/ethers.js/blob/master/CHANGELOG.md).


Installing
----------

**node.js**

```
/home/ricmoo/some_project> npm install --save ethers
```

**browser (UMD)**

```
<script src="https://cdn.ethers.io/lib/ethers-5.0.umd.min.js" type="text/javascript">
</script>
```

**browser (ESM)**

```
<script type="module">
    import { ethers } from "https://cdn.ethers.io/lib/ethers-5.0.umd.min.js";
</script>
```


Documentation
-------------

Browse the [documentation](https://docs.ethers.io/v5/) online:

- [Getting Started](https://docs.ethers.io/v5/getting-started/)
- [Full API Documentation](https://docs.ethers.io/v5/api/)
- [Various Ethereum Articles](https://blog.ricmoo.com/)

Or browse the entire documentation as a [single page](https://docs.ethers.io/v5/single-page/) to make searching easier.


Ancillary Packages
------------------

These are a number of packages not included in the umbrella `ethers` npm package, and
additional packages are always being added. Often these packages are for specific
use-cases, so rather than adding them to the umbrella package, they are added as
ancillary packages, which can be included by those who need them, while not bloating
everyone else with packages they do not need.

We will keep a list of useful packages here.

- `@ethersproject/experimental` ([documentation](https://docs.ethers.io))
- `@ethersproject/cli` ([documentation](https://docs.ethers.io))
- `@ethersproject/hardware-wallets` ([documentation](https://docs.ethers.io))


Sponsors
--------

Support the ethers project by [becoming a sponsor](https://ethers.org/sponsoring.html).
Get your logo added below with a link to your website (Gold and Unobtainium tiers)
and on the [ethers.org](https://ethers.org) website.

Huge thanks to our sponsors! `<3 <3`

<a href="https://ethers.org/sponsors/tally-link" target="_blank"><img src="https://ethers.org/sponsors/tally-readme.svg"></a>


License
-------

MIT License (including **all** dependencies).

