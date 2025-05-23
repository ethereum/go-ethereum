/**
 * The `web3-eth-personal` package allows you to interact with the Ethereum node’s accounts.
 *
 * **_NOTE:_**  Many of these functions send sensitive information like passwords. Never call these functions over a unsecured Websocket or HTTP provider, as your password will be sent in plain text!
 *
 * import Personal from 'web3-eth-personal';
 *
 * const personal = new Personal('http://localhost:8545');
 *
 * or using the web3 umbrella package
 *
 * import Personal from 'web3-eth-personal';
 * const web3 = new Web3('http://localhost:8545');
 * // web3.eth.personal
 */
/**
 * This comment _supports3_ [Markdown](https://marked.js.org/)
 */
import { Personal } from './personal.js';
export * from './personal.js';
export default Personal;
