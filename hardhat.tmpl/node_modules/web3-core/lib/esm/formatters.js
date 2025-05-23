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
import { FormatterError } from 'web3-errors';
import { Iban } from 'web3-eth-iban';
import { BlockTags, } from 'web3-types';
import { fromUtf8, hexToNumber, hexToNumberString, isAddress, isHexStrict, mergeDeep, numberToHex, sha3Raw, toChecksumAddress, toNumber, toUtf8, utf8ToHex, } from 'web3-utils';
import { isBlockTag, isHex, isNullish } from 'web3-validator';
/* eslint-disable deprecation/deprecation */
/**
 * @deprecated Use format function from web3-utils package instead
 * Will format the given storage key array values to hex strings.
 */
export const inputStorageKeysFormatter = (keys) => keys.map(num => numberToHex(num));
/**
 * @deprecated Use format function from web3-utils package instead
 * Will format the given proof response from the node.
 */
export const outputProofFormatter = (proof) => ({
    address: toChecksumAddress(proof.address),
    nonce: hexToNumberString(proof.nonce),
    balance: hexToNumberString(proof.balance),
});
/**
 * @deprecated Use format function from web3-utils package instead
 * Should the format output to a big number
 */
export const outputBigIntegerFormatter = (number) => toNumber(number);
/**
 * @deprecated Use format function from web3-utils package instead
 * Returns the given block number as hex string or the predefined block number 'latest', 'pending', 'earliest', 'genesis'
 */
export const inputBlockNumberFormatter = (blockNumber) => {
    if (isNullish(blockNumber)) {
        return undefined;
    }
    if (typeof blockNumber === 'string' && isBlockTag(blockNumber)) {
        return blockNumber;
    }
    if (blockNumber === 'genesis') {
        return '0x0';
    }
    if (typeof blockNumber === 'string' && isHexStrict(blockNumber)) {
        return blockNumber.toLowerCase();
    }
    return numberToHex(blockNumber);
};
/**
 * @deprecated Use format function from web3-utils package instead
 * Returns the given block number as hex string or does return the defaultBlock property of the current module
 */
export const inputDefaultBlockNumberFormatter = (blockNumber, defaultBlock) => {
    if (!blockNumber) {
        return inputBlockNumberFormatter(defaultBlock);
    }
    return inputBlockNumberFormatter(blockNumber);
};
/**
 * @deprecated Use format function from web3-utils package instead
 * @param address
 */
export const inputAddressFormatter = (address) => {
    if (Iban.isValid(address) && Iban.isDirect(address)) {
        const iban = new Iban(address);
        return iban.toAddress().toLowerCase();
    }
    if (isAddress(address)) {
        return `0x${address.toLowerCase().replace('0x', '')}`;
    }
    throw new FormatterError(`Provided address ${address} is invalid, the capitalization checksum test failed, or it's an indirect IBAN address which can't be converted.`);
};
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a transaction and converts all values to HEX
 */
export const txInputOptionsFormatter = (options) => {
    var _a;
    const modifiedOptions = Object.assign({}, options);
    if (options.to) {
        // it might be contract creation
        modifiedOptions.to = inputAddressFormatter(options.to);
    }
    if (options.data && options.input) {
        throw new FormatterError('You can\'t have "data" and "input" as properties of transactions at the same time, please use either "data" or "input" instead.');
    }
    if (!options.input && options.data) {
        modifiedOptions.input = options.data;
        delete modifiedOptions.data;
    }
    if (options.input && !options.input.startsWith('0x')) {
        modifiedOptions.input = `0x${options.input}`;
    }
    if (modifiedOptions.input && !isHexStrict(modifiedOptions.input)) {
        throw new FormatterError('The input field must be HEX encoded data.');
    }
    // allow both
    if (options.gas || options.gasLimit) {
        modifiedOptions.gas = toNumber((_a = options.gas) !== null && _a !== void 0 ? _a : options.gasLimit);
    }
    if (options.maxPriorityFeePerGas || options.maxFeePerGas) {
        delete modifiedOptions.gasPrice;
    }
    ['gasPrice', 'gas', 'value', 'maxPriorityFeePerGas', 'maxFeePerGas', 'nonce', 'chainId']
        .filter(key => !isNullish(modifiedOptions[key]))
        .forEach(key => {
        modifiedOptions[key] = numberToHex(modifiedOptions[key]);
    });
    return modifiedOptions;
};
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a transaction and converts all values to HEX
 */
export const inputCallFormatter = (options, defaultAccount) => {
    var _a;
    const opts = txInputOptionsFormatter(options);
    const from = (_a = opts.from) !== null && _a !== void 0 ? _a : defaultAccount;
    if (from) {
        opts.from = inputAddressFormatter(from);
    }
    return opts;
};
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a transaction and converts all values to HEX
 */
export const inputTransactionFormatter = (options, defaultAccount) => {
    var _a;
    const opts = txInputOptionsFormatter(options);
    // check from, only if not number, or object
    if (!(typeof opts.from === 'number') && !(!!opts.from && typeof opts.from === 'object')) {
        opts.from = (_a = opts.from) !== null && _a !== void 0 ? _a : defaultAccount;
        if (!options.from && !(typeof options.from === 'number')) {
            throw new FormatterError('The send transactions "from" field must be defined!');
        }
        opts.from = inputAddressFormatter(options.from);
    }
    return opts;
};
/**
 * @deprecated Use format function from web3-utils package instead
 * Hex encodes the data passed to eth_sign and personal_sign
 */
export const inputSignFormatter = (data) => (isHexStrict(data) ? data : utf8ToHex(data));
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a transaction to its proper values
 * @function outputTransactionFormatter
 */
export const outputTransactionFormatter = (tx) => {
    const modifiedTx = Object.assign({}, tx);
    if (tx.blockNumber) {
        modifiedTx.blockNumber = hexToNumber(tx.blockNumber);
    }
    if (tx.transactionIndex) {
        modifiedTx.transactionIndex = hexToNumber(tx.transactionIndex);
    }
    modifiedTx.nonce = hexToNumber(tx.nonce);
    modifiedTx.gas = hexToNumber(tx.gas);
    if (tx.gasPrice) {
        modifiedTx.gasPrice = outputBigIntegerFormatter(tx.gasPrice);
    }
    if (tx.maxFeePerGas) {
        modifiedTx.maxFeePerGas = outputBigIntegerFormatter(tx.maxFeePerGas);
    }
    if (tx.maxPriorityFeePerGas) {
        modifiedTx.maxPriorityFeePerGas = outputBigIntegerFormatter(tx.maxPriorityFeePerGas);
    }
    if (tx.type) {
        modifiedTx.type = hexToNumber(tx.type);
    }
    modifiedTx.value = outputBigIntegerFormatter(tx.value);
    if (tx.to && isAddress(tx.to)) {
        // tx.to could be `0x0` or `null` while contract creation
        modifiedTx.to = toChecksumAddress(tx.to);
    }
    else {
        modifiedTx.to = undefined; // set to `null` if invalid address
    }
    if (tx.from) {
        modifiedTx.from = toChecksumAddress(tx.from);
    }
    return modifiedTx;
};
/**
 * @deprecated Use format function from web3-utils package instead
 * @param topic
 */
// To align with specification we use the type "null" here
// eslint-disable-next-line @typescript-eslint/ban-types
export const inputTopicFormatter = (topic) => {
    // Using "null" value intentionally for validation
    // eslint-disable-next-line no-null/no-null
    if (isNullish(topic))
        return null;
    const value = String(topic);
    return isHex(value) ? value : fromUtf8(value);
};
/**
 * @deprecated Use format function from web3-utils package instead
 * @param filter
 */
export const inputLogFormatter = (filter) => {
    var _a;
    const val = isNullish(filter)
        ? {}
        : mergeDeep({}, filter);
    // If options !== undefined, don't blow out existing data
    if (isNullish(val.fromBlock)) {
        val.fromBlock = BlockTags.LATEST;
    }
    val.fromBlock = inputBlockNumberFormatter(val.fromBlock);
    if (!isNullish(val.toBlock)) {
        val.toBlock = inputBlockNumberFormatter(val.toBlock);
    }
    // make sure topics, get converted to hex
    val.topics = (_a = val.topics) !== null && _a !== void 0 ? _a : [];
    val.topics = val.topics.map(topic => Array.isArray(topic)
        ? topic.map(inputTopicFormatter)
        : inputTopicFormatter(topic));
    if (val.address) {
        val.address = Array.isArray(val.address)
            ? val.address.map(addr => inputAddressFormatter(addr))
            : inputAddressFormatter(val.address);
    }
    return val;
};
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a log
 * @function outputLogFormatter
 */
export const outputLogFormatter = (log) => {
    const modifiedLog = Object.assign({}, log);
    const logIndex = typeof log.logIndex === 'string'
        ? log.logIndex
        : numberToHex(log.logIndex);
    // generate a custom log id
    if (typeof log.blockHash === 'string' && typeof log.transactionHash === 'string') {
        const shaId = sha3Raw(`${log.blockHash.replace('0x', '')}${log.transactionHash.replace('0x', '')}${logIndex.replace('0x', '')}`);
        modifiedLog.id = `log_${shaId.replace('0x', '').slice(0, 8)}`;
    }
    else if (!log.id) {
        modifiedLog.id = undefined;
    }
    if (log.blockNumber && isHexStrict(log.blockNumber)) {
        modifiedLog.blockNumber = hexToNumber(log.blockNumber);
    }
    if (log.transactionIndex && isHexStrict(log.transactionIndex)) {
        modifiedLog.transactionIndex = hexToNumber(log.transactionIndex);
    }
    if (log.logIndex && isHexStrict(log.logIndex)) {
        modifiedLog.logIndex = hexToNumber(log.logIndex);
    }
    if (log.address) {
        modifiedLog.address = toChecksumAddress(log.address);
    }
    return modifiedLog;
};
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a transaction receipt to its proper values
 */
export const outputTransactionReceiptFormatter = (receipt) => {
    if (typeof receipt !== 'object') {
        throw new FormatterError(`Received receipt is invalid: ${String(receipt)}`);
    }
    const modifiedReceipt = Object.assign({}, receipt);
    if (receipt.blockNumber) {
        modifiedReceipt.blockNumber = hexToNumber(receipt.blockNumber);
    }
    if (receipt.transactionIndex) {
        modifiedReceipt.transactionIndex = hexToNumber(receipt.transactionIndex);
    }
    modifiedReceipt.cumulativeGasUsed = hexToNumber(receipt.cumulativeGasUsed);
    modifiedReceipt.gasUsed = hexToNumber(receipt.gasUsed);
    if (receipt.logs && Array.isArray(receipt.logs)) {
        modifiedReceipt.logs = receipt.logs.map(outputLogFormatter);
    }
    if (receipt.effectiveGasPrice) {
        modifiedReceipt.effectiveGasPrice = hexToNumber(receipt.effectiveGasPrice);
    }
    if (receipt.contractAddress) {
        modifiedReceipt.contractAddress = toChecksumAddress(receipt.contractAddress);
    }
    if (receipt.status) {
        modifiedReceipt.status = Boolean(parseInt(receipt.status, 10));
    }
    return modifiedReceipt;
};
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a block to its proper values
 * @function outputBlockFormatter
 */
export const outputBlockFormatter = (block) => {
    const modifiedBlock = Object.assign({}, block);
    // transform to number
    modifiedBlock.gasLimit = hexToNumber(block.gasLimit);
    modifiedBlock.gasUsed = hexToNumber(block.gasUsed);
    modifiedBlock.size = hexToNumber(block.size);
    modifiedBlock.timestamp = hexToNumber(block.timestamp);
    if (block.number) {
        modifiedBlock.number = hexToNumber(block.number);
    }
    if (block.difficulty) {
        modifiedBlock.difficulty = outputBigIntegerFormatter(block.difficulty);
    }
    if (block.totalDifficulty) {
        modifiedBlock.totalDifficulty = outputBigIntegerFormatter(block.totalDifficulty);
    }
    if (block.transactions && Array.isArray(block.transactions)) {
        modifiedBlock.transactions = block.transactions.map(outputTransactionFormatter);
    }
    if (block.miner) {
        modifiedBlock.miner = toChecksumAddress(block.miner);
    }
    if (block.baseFeePerGas) {
        modifiedBlock.baseFeePerGas = outputBigIntegerFormatter(block.baseFeePerGas);
    }
    return modifiedBlock;
};
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a whisper post and converts all values to HEX
 */
export const inputPostFormatter = (post) => {
    var _a;
    const modifiedPost = Object.assign({}, post);
    if (post.ttl) {
        modifiedPost.ttl = numberToHex(post.ttl);
    }
    if (post.workToProve) {
        modifiedPost.workToProve = numberToHex(post.workToProve);
    }
    if (post.priority) {
        modifiedPost.priority = numberToHex(post.priority);
    }
    // fallback
    if (post.topics && !Array.isArray(post.topics)) {
        modifiedPost.topics = post.topics ? [post.topics] : [];
    }
    // format the following options
    modifiedPost.topics = (_a = modifiedPost.topics) === null || _a === void 0 ? void 0 : _a.map(topic => topic.startsWith('0x') ? topic : fromUtf8(topic));
    return modifiedPost;
};
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a received post message
 * @function outputPostFormatter
 */
export const outputPostFormatter = (post) => {
    var _a;
    const modifiedPost = Object.assign({}, post);
    if (post.expiry) {
        modifiedPost.expiry = hexToNumber(post.expiry);
    }
    if (post.sent) {
        modifiedPost.sent = hexToNumber(post.sent);
    }
    if (post.ttl) {
        modifiedPost.ttl = hexToNumber(post.ttl);
    }
    if (post.workProved) {
        modifiedPost.workProved = hexToNumber(post.workProved);
    }
    // post.payloadRaw = post.payload;
    // post.payload = utils.hexToAscii(post.payload);
    // if (utils.isJson(post.payload)) {
    //     post.payload = JSON.parse(post.payload);
    // }
    // format the following options
    if (!post.topics) {
        modifiedPost.topics = [];
    }
    modifiedPost.topics = (_a = modifiedPost.topics) === null || _a === void 0 ? void 0 : _a.map(toUtf8);
    return modifiedPost;
};
/**
 * @deprecated Use format function from web3-utils package instead
 */
export const outputSyncingFormatter = (result) => {
    const modifiedResult = Object.assign({}, result);
    modifiedResult.startingBlock = hexToNumber(result.startingBlock);
    modifiedResult.currentBlock = hexToNumber(result.currentBlock);
    modifiedResult.highestBlock = hexToNumber(result.highestBlock);
    if (result.knownStates) {
        modifiedResult.knownStates = hexToNumber(result.knownStates);
    }
    if (result.pulledStates) {
        modifiedResult.pulledStates = hexToNumber(result.pulledStates);
    }
    return modifiedResult;
};
//# sourceMappingURL=formatters.js.map