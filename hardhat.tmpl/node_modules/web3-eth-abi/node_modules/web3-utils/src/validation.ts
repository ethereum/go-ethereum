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

import { InvalidBlockError } from 'web3-errors';
import {
	checkAddressCheckSum as checkAddressCheckSumValidator,
	isAddress as isAddressValidator,
	isBlockTag,
	isBloom as isBloomValidator,
	isContractAddressInBloom as isContractAddressInBloomValidator,
	isHex as isHexValidator,
	isHexStrict as isHexStrictValidator,
	isInBloom as isInBloomValidator,
	isNullish as isNullishValidator,
	isTopic as isTopicValidator,
	isTopicInBloom as isTopicInBloomValidator,
	isUserEthereumAddressInBloom as isUserEthereumAddressInBloomValidator,
} from 'web3-validator';
import { BlockNumberOrTag, BlockTags, ContractInitOptions } from 'web3-types';

/**
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export const isHexStrict = isHexStrictValidator;

/**
 * returns true if input is a hexstring, number or bigint
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export const isHex = isHexValidator;

/**
 * Checks the checksum of a given address. Will also return false on non-checksum addresses.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export const checkAddressCheckSum = checkAddressCheckSumValidator;

/**
 * Checks if a given string is a valid Ethereum address. It will also check the checksum, if the address has upper and lowercase letters.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export const isAddress = isAddressValidator;

/**
 * Returns true if the bloom is a valid bloom
 * https://github.com/joshstevens19/ethereum-bloom-filters/blob/fbeb47b70b46243c3963fe1c2988d7461ef17236/src/index.ts#L7
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export const isBloom = isBloomValidator;

/**
 * Returns true if the value is part of the given bloom
 * note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export const isInBloom = isInBloomValidator;

/**
 * Returns true if the ethereum users address is part of the given bloom note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export const isUserEthereumAddressInBloom = isUserEthereumAddressInBloomValidator;

/**
 * Returns true if the contract address is part of the given bloom.
 * note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export const isContractAddressInBloom = isContractAddressInBloomValidator;

/**
 * Checks if its a valid topic
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export const isTopic = isTopicValidator;

/**
 * Returns true if the topic is part of the given bloom.
 * note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export const isTopicInBloom = isTopicInBloomValidator;

/**
 * Compares between block A and block B
 * @param blockA - Block number or string
 * @param blockB - Block number or string
 *
 * @returns - Returns -1 if a \< b, returns 1 if a \> b and returns 0 if a == b
 *
 * @example
 * ```ts
 * console.log(web3.utils.compareBlockNumbers('latest', 'pending'));
 * > -1
 *
 * console.log(web3.utils.compareBlockNumbers(12, 11));
 * > 1
 * ```
 */
export const compareBlockNumbers = (blockA: BlockNumberOrTag, blockB: BlockNumberOrTag) => {
	const isABlockTag = typeof blockA === 'string' && isBlockTag(blockA);
	const isBBlockTag = typeof blockB === 'string' && isBlockTag(blockB);

	if (
		blockA === blockB ||
		((blockA === 'earliest' || blockA === 0) && (blockB === 'earliest' || blockB === 0)) // only exception compare blocktag with number
	) {
		return 0;
	}
	if (blockA === 'earliest') {
		return -1;
	}
	if (blockB === 'earliest') {
		return 1;
	}

	if (isABlockTag && isBBlockTag) {
		// Increasing order:  earliest, finalized , safe, latest, pending
		const tagsOrder = {
			[BlockTags.EARLIEST as string]: 1,
			[BlockTags.FINALIZED as string]: 2,
			[BlockTags.SAFE as string]: 3,
			[BlockTags.LATEST as string]: 4,
			[BlockTags.PENDING as string]: 5,
		};

		if (tagsOrder[blockA] < tagsOrder[blockB]) {
			return -1;
		}

		return 1;
	}
	if ((isABlockTag && !isBBlockTag) || (!isABlockTag && isBBlockTag)) {
		throw new InvalidBlockError('Cannot compare blocktag with provided non-blocktag input.');
	}

	const bigIntA = BigInt(blockA);
	const bigIntB = BigInt(blockB);

	if (bigIntA < bigIntB) {
		return -1;
	}
	if (bigIntA === bigIntB) {
		return 0;
	}
	return 1;
};

export const isContractInitOptions = (options: unknown): options is ContractInitOptions =>
	typeof options === 'object' &&
	!isNullishValidator(options) &&
	Object.keys(options).length !== 0 &&
	[
		'input',
		'data',
		'from',
		'gas',
		'gasPrice',
		'gasLimit',
		'address',
		'jsonInterface',
		'syncWithContext',
		'dataInputFill',
	].some(key => key in options);

export const isNullish = isNullishValidator;
