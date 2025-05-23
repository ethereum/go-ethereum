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
