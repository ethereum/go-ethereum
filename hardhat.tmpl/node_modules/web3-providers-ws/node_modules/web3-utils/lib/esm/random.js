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
 * @module Utils
 */
import { getRandomBytesSync } from 'ethereum-cryptography/random.js';
import { bytesToHex } from './converters.js';
/**
 * Returns a random byte array by the given bytes size
 * @param size - The size of the random byte array returned
 * @returns - random byte array
 *
 * @example
 * ```ts
 * console.log(web3.utils.randomBytes(32));
 * > Uint8Array(32) [
 *       93, 172, 226,  32,  33, 176, 156, 156,
 *       182,  30, 240,   2,  69,  96, 174, 197,
 *       33, 136, 194, 241, 197, 156, 110, 111,
 *       66,  87,  17,  88,  67,  48, 245, 183
 *    ]
 * ```
 */
export const randomBytes = (size) => getRandomBytesSync(size);
/**
 * Returns a random hex string by the given bytes size
 * @param byteSize - The size of the random hex string returned
 * @returns - random hex string
 *
 * ```ts
 * console.log(web3.utils.randomHex(32));
 * > 0x139f5b88b72a25eab053d3b57fe1f8a9dbc62a526b1cb1774d0d7db1c3e7ce9e
 * ```
 */
export const randomHex = (byteSize) => bytesToHex(randomBytes(byteSize));
//# sourceMappingURL=random.js.map