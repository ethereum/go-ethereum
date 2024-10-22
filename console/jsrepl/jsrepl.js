// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Pull in a few packages needed for the console
const ethers = require('ethers');
const repl   = require('pretty-repl');
const util   = require('util');
const fs     = require('fs');

import chalk from 'chalk';

// Connect to the requested node and create an API wrapper
function dial(url) {
    if (url.startsWith('ws://') || url.startsWith('wss://')) {
        return new ethers.WebSocketProvider(url);
    } else if (url.startsWith('http://') || url.startsWith('https://')) {
        return new ethers.JsonRpcProvider(url);
    } else {
        return new ethers.IpcSocketProvider(url);
    }
}
const pubapi = dial(process.argv[2]);

// Ethers.js is opinionated, swallowing certain fields from responses. Whilst
// that is fine for the user API, for the Geth console APIs, we would like to
// have everything exposed for cleaner testing. Create a secondary API object
// that injects all missing fields back (sorry ethers).
const dbgapi = dial(process.argv[2]);

const oldWrapBlock = dbgapi._wrapBlock.bind(dbgapi);
dbgapi._wrapBlock = (value, format) => {
    const block = oldWrapBlock(value, format);
    Object.keys(value).forEach((key) => {
        if (!(key in block)) {
            block[key] = value[key]; // Transform anything here if need be
        }
    });
    return block;
};

// Collect all the custom methods we want to expose
const context = {
    ethers: ethers, // Expose ethers for power users
    client: pubapi, // Expose the original client provider
    hooked: dbgapi, // Expose the hooked client provider

    // Define the Geth specific specialized methods (TODO)
    eth: {
        getBlock: dbgapi.getBlock.bind(dbgapi),
    }
};

// Print some startup headers
const welcome = async () => {
    const client  = await pubapi.send("web3_clientVersion");
    const block   = await pubapi.getBlock();
    const modules = await pubapi.send("rpc_modules");

    console.log(`Welcome to the Geth console!\n`);

    console.log(`Geth: ${chalk.green("go-ethereum")} ${chalk.yellow("v" + process.env.TINYGETH_CONSOLE_VERSION)}`);
    console.log(`REPL: ${chalk.green("     nodejs")} ${chalk.yellow(process.version)}`);
    console.log(`Web3: ${chalk.green("     ethers")} ${chalk.yellow("v" + ethers.version)}\n`);

    console.log(`Attached: ${client}`);
    console.log(`At block: ${block.number} (${new Date(1000 * Number(block.timestamp))})`);
    console.log(` Exposed: ${Object.keys(modules).join(' ')}\n`);

    console.log(chalk.grey("• The web3 library uses promises, you need to await appropriately."));
    console.log(chalk.grey("• The usual Geth API methods are exposed to the root namespace."));
    console.log(chalk.grey("• The web3 library is exposed in full in the `ethers` field."));
    console.log(chalk.grey("• The web3 connection is exposed via the `client` field."));
    console.log(chalk.grey("• The `hooked` client exposes all data from the RPC.\n"));
};

// Start the REPL server and inject all context into it
const startup = async () => {
    const server = repl.start({
        ignoreUndefined: true,
        useGlobal: true,

        prompt: chalk.green('→ '),
        writer: (output) => {
            if (output && typeof output.stack === 'string' && typeof output.message === 'string') {
                return chalk.red(output.stack || output.message);
            }
            return chalk.gray(util.inspect(output, {colors: true, depth: null}));
        }
    });
    server.on('exit', () => {
        process.exit(); // Force tear down all resources
    });
    Object.assign(server.context, context);
};

welcome().then(() => startup());

// REPL started and entire script evaluated, self-destruct. This is a weird one
// but since we can't delete from Go (process reown), this is the only remaining
// place to clean up the script.
fs.unlinkSync(process.argv[1]);
