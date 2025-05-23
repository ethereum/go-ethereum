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
