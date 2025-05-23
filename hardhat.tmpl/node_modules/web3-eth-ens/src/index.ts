/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/

/**
 * The `web3.eth.ens` functions let you interact with ENS. We recommend reading the [ENS documentation](https://docs.ens.domains/) to get deeper insights about the internals of the name service.
 *
 * ## Breaking Changes
 *
 * -   All the API level interfaces returning or accepting `null` in 1.x, use `undefined` in 4.x.
 * -   Functions don't accept a callback anymore.
 * -   Functions that accepted an optional `TransactionConfig` as the last argument, now accept an optional `NonPayableCallOptions`. See `web3-eth-contract` package for more details.
 * -   Removed all non-read methods. If you need modifing resolver or registry, please use https://www.npmjs.com/package/@ensdomains/ensjs
 */
/**
 * This comment _supports3_ [Markdown](https://marked.js.org/)
 */
import { registryAddresses } from './config.js';

export * from './ens.js';
export { registryAddresses };
