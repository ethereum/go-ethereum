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

import { BlockTags } from 'web3-types';
import { isUInt } from './numbers.js';

export const isBlockNumber = (value: string | number | bigint): boolean => isUInt(value);

/**
 * Returns true if the given blockNumber is 'latest', 'pending', 'earliest, 'safe' or 'finalized'
 */
export const isBlockTag = (value: string) => Object.values(BlockTags).includes(value as BlockTags);

/**
 * Returns true if given value is valid hex string and not negative, or is a valid BlockTag
 */
export const isBlockNumberOrTag = (value: string | number | bigint) =>
	isBlockTag(value as string) || isBlockNumber(value);
