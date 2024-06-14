#!/usr/bin/env node

const c = require('chalk');
const emoji = require('node-emoji');

const title = `
:warning:   ${c.red.bold('solidity-coverage >= 0.7.0 does not use "testrpc-sc"')}  :warning:
${c.bold('===========================================================')}`;

const info = `
Instead, you can use any ganache version you'd like & configure it as you wish
via the .solcover.js options. (It also comes with a default version for easy use.)

> ${c.bold('Launching the client independently of the coverage tool is no longer supported.')}
> See github.com/sc-forks/solidity-coverage for help with configuration.`;

const thanks = `
Thanks! - sc-forks
`;


const msg = `
${title}
${info}
${c.green.bold(thanks)}
`
console.log(emoji.emojify(msg));

process.exit(1);
