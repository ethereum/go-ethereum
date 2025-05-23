"use strict";
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.isNullish = exports.isContractInitOptions = exports.compareBlockNumbers = exports.isTopicInBloom = exports.isTopic = exports.isContractAddressInBloom = exports.isUserEthereumAddressInBloom = exports.isInBloom = exports.isBloom = exports.isAddress = exports.checkAddressCheckSum = exports.isHex = exports.isHexStrict = void 0;
/**
 * @module Utils
 */
const web3_errors_1 = require("web3-errors");
const web3_validator_1 = require("web3-validator");
const web3_types_1 = require("web3-types");
/**
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
exports.isHexStrict = web3_validator_1.isHexStrict;
/**
 * returns true if input is a hexstring, number or bigint
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
exports.isHex = web3_validator_1.isHex;
/**
 * Checks the checksum of a given address. Will also return false on non-checksum addresses.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
exports.checkAddressCheckSum = web3_validator_1.checkAddressCheckSum;
/**
 * Checks if a given string is a valid Ethereum address. It will also check the checksum, if the address has upper and lowercase letters.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
exports.isAddress = web3_validator_1.isAddress;
/**
 * Returns true if the bloom is a valid bloom
 * https://github.com/joshstevens19/ethereum-bloom-filters/blob/fbeb47b70b46243c3963fe1c2988d7461ef17236/src/index.ts#L7
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
exports.isBloom = web3_validator_1.isBloom;
/**
 * Returns true if the value is part of the given bloom
 * note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
exports.isInBloom = web3_validator_1.isInBloom;
/**
 * Returns true if the ethereum users address is part of the given bloom note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
exports.isUserEthereumAddressInBloom = web3_validator_1.isUserEthereumAddressInBloom;
/**
 * Returns true if the contract address is part of the given bloom.
 * note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
exports.isContractAddressInBloom = web3_validator_1.isContractAddressInBloom;
/**
 * Checks if its a valid topic
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
exports.isTopic = web3_validator_1.isTopic;
/**
 * Returns true if the topic is part of the given bloom.
 * note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
exports.isTopicInBloom = web3_validator_1.isTopicInBloom;
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
const compareBlockNumbers = (blockA, blockB) => {
    const isABlockTag = typeof blockA === 'string' && (0, web3_validator_1.isBlockTag)(blockA);
    const isBBlockTag = typeof blockB === 'string' && (0, web3_validator_1.isBlockTag)(blockB);
    if (blockA === blockB ||
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
            [web3_types_1.BlockTags.EARLIEST]: 1,
            [web3_types_1.BlockTags.FINALIZED]: 2,
            [web3_types_1.BlockTags.SAFE]: 3,
            [web3_types_1.BlockTags.LATEST]: 4,
            [web3_types_1.BlockTags.PENDING]: 5,
        };
        if (tagsOrder[blockA] < tagsOrder[blockB]) {
            return -1;
        }
        return 1;
    }
    if ((isABlockTag && !isBBlockTag) || (!isABlockTag && isBBlockTag)) {
        throw new web3_errors_1.InvalidBlockError('Cannot compare blocktag with provided non-blocktag input.');
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
exports.compareBlockNumbers = compareBlockNumbers;
const isContractInitOptions = (options) => typeof options === 'object' &&
    !(0, web3_validator_1.isNullish)(options) &&
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
exports.isContractInitOptions = isContractInitOptions;
exports.isNullish = web3_validator_1.isNullish;
//# sourceMappingURL=validation.js.map