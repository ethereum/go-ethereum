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
exports.outputSyncingFormatter = exports.outputPostFormatter = exports.inputPostFormatter = exports.outputBlockFormatter = exports.outputTransactionReceiptFormatter = exports.outputLogFormatter = exports.inputLogFormatter = exports.inputTopicFormatter = exports.outputTransactionFormatter = exports.inputSignFormatter = exports.inputTransactionFormatter = exports.inputCallFormatter = exports.txInputOptionsFormatter = exports.inputAddressFormatter = exports.inputDefaultBlockNumberFormatter = exports.inputBlockNumberFormatter = exports.outputBigIntegerFormatter = exports.outputProofFormatter = exports.inputStorageKeysFormatter = void 0;
const web3_errors_1 = require("web3-errors");
const web3_eth_iban_1 = require("web3-eth-iban");
const web3_types_1 = require("web3-types");
const web3_utils_1 = require("web3-utils");
const web3_validator_1 = require("web3-validator");
/* eslint-disable deprecation/deprecation */
/**
 * @deprecated Use format function from web3-utils package instead
 * Will format the given storage key array values to hex strings.
 */
const inputStorageKeysFormatter = (keys) => keys.map(num => (0, web3_utils_1.numberToHex)(num));
exports.inputStorageKeysFormatter = inputStorageKeysFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Will format the given proof response from the node.
 */
const outputProofFormatter = (proof) => ({
    address: (0, web3_utils_1.toChecksumAddress)(proof.address),
    nonce: (0, web3_utils_1.hexToNumberString)(proof.nonce),
    balance: (0, web3_utils_1.hexToNumberString)(proof.balance),
});
exports.outputProofFormatter = outputProofFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Should the format output to a big number
 */
const outputBigIntegerFormatter = (number) => (0, web3_utils_1.toNumber)(number);
exports.outputBigIntegerFormatter = outputBigIntegerFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Returns the given block number as hex string or the predefined block number 'latest', 'pending', 'earliest', 'genesis'
 */
const inputBlockNumberFormatter = (blockNumber) => {
    if ((0, web3_validator_1.isNullish)(blockNumber)) {
        return undefined;
    }
    if (typeof blockNumber === 'string' && (0, web3_validator_1.isBlockTag)(blockNumber)) {
        return blockNumber;
    }
    if (blockNumber === 'genesis') {
        return '0x0';
    }
    if (typeof blockNumber === 'string' && (0, web3_utils_1.isHexStrict)(blockNumber)) {
        return blockNumber.toLowerCase();
    }
    return (0, web3_utils_1.numberToHex)(blockNumber);
};
exports.inputBlockNumberFormatter = inputBlockNumberFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Returns the given block number as hex string or does return the defaultBlock property of the current module
 */
const inputDefaultBlockNumberFormatter = (blockNumber, defaultBlock) => {
    if (!blockNumber) {
        return (0, exports.inputBlockNumberFormatter)(defaultBlock);
    }
    return (0, exports.inputBlockNumberFormatter)(blockNumber);
};
exports.inputDefaultBlockNumberFormatter = inputDefaultBlockNumberFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * @param address
 */
const inputAddressFormatter = (address) => {
    if (web3_eth_iban_1.Iban.isValid(address) && web3_eth_iban_1.Iban.isDirect(address)) {
        const iban = new web3_eth_iban_1.Iban(address);
        return iban.toAddress().toLowerCase();
    }
    if ((0, web3_utils_1.isAddress)(address)) {
        return `0x${address.toLowerCase().replace('0x', '')}`;
    }
    throw new web3_errors_1.FormatterError(`Provided address ${address} is invalid, the capitalization checksum test failed, or it's an indirect IBAN address which can't be converted.`);
};
exports.inputAddressFormatter = inputAddressFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a transaction and converts all values to HEX
 */
const txInputOptionsFormatter = (options) => {
    var _a;
    const modifiedOptions = Object.assign({}, options);
    if (options.to) {
        // it might be contract creation
        modifiedOptions.to = (0, exports.inputAddressFormatter)(options.to);
    }
    if (options.data && options.input) {
        throw new web3_errors_1.FormatterError('You can\'t have "data" and "input" as properties of transactions at the same time, please use either "data" or "input" instead.');
    }
    if (!options.input && options.data) {
        modifiedOptions.input = options.data;
        delete modifiedOptions.data;
    }
    if (options.input && !options.input.startsWith('0x')) {
        modifiedOptions.input = `0x${options.input}`;
    }
    if (modifiedOptions.input && !(0, web3_utils_1.isHexStrict)(modifiedOptions.input)) {
        throw new web3_errors_1.FormatterError('The input field must be HEX encoded data.');
    }
    // allow both
    if (options.gas || options.gasLimit) {
        modifiedOptions.gas = (0, web3_utils_1.toNumber)((_a = options.gas) !== null && _a !== void 0 ? _a : options.gasLimit);
    }
    if (options.maxPriorityFeePerGas || options.maxFeePerGas) {
        delete modifiedOptions.gasPrice;
    }
    ['gasPrice', 'gas', 'value', 'maxPriorityFeePerGas', 'maxFeePerGas', 'nonce', 'chainId']
        .filter(key => !(0, web3_validator_1.isNullish)(modifiedOptions[key]))
        .forEach(key => {
        modifiedOptions[key] = (0, web3_utils_1.numberToHex)(modifiedOptions[key]);
    });
    return modifiedOptions;
};
exports.txInputOptionsFormatter = txInputOptionsFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a transaction and converts all values to HEX
 */
const inputCallFormatter = (options, defaultAccount) => {
    var _a;
    const opts = (0, exports.txInputOptionsFormatter)(options);
    const from = (_a = opts.from) !== null && _a !== void 0 ? _a : defaultAccount;
    if (from) {
        opts.from = (0, exports.inputAddressFormatter)(from);
    }
    return opts;
};
exports.inputCallFormatter = inputCallFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a transaction and converts all values to HEX
 */
const inputTransactionFormatter = (options, defaultAccount) => {
    var _a;
    const opts = (0, exports.txInputOptionsFormatter)(options);
    // check from, only if not number, or object
    if (!(typeof opts.from === 'number') && !(!!opts.from && typeof opts.from === 'object')) {
        opts.from = (_a = opts.from) !== null && _a !== void 0 ? _a : defaultAccount;
        if (!options.from && !(typeof options.from === 'number')) {
            throw new web3_errors_1.FormatterError('The send transactions "from" field must be defined!');
        }
        opts.from = (0, exports.inputAddressFormatter)(options.from);
    }
    return opts;
};
exports.inputTransactionFormatter = inputTransactionFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Hex encodes the data passed to eth_sign and personal_sign
 */
const inputSignFormatter = (data) => ((0, web3_utils_1.isHexStrict)(data) ? data : (0, web3_utils_1.utf8ToHex)(data));
exports.inputSignFormatter = inputSignFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a transaction to its proper values
 * @function outputTransactionFormatter
 */
const outputTransactionFormatter = (tx) => {
    const modifiedTx = Object.assign({}, tx);
    if (tx.blockNumber) {
        modifiedTx.blockNumber = (0, web3_utils_1.hexToNumber)(tx.blockNumber);
    }
    if (tx.transactionIndex) {
        modifiedTx.transactionIndex = (0, web3_utils_1.hexToNumber)(tx.transactionIndex);
    }
    modifiedTx.nonce = (0, web3_utils_1.hexToNumber)(tx.nonce);
    modifiedTx.gas = (0, web3_utils_1.hexToNumber)(tx.gas);
    if (tx.gasPrice) {
        modifiedTx.gasPrice = (0, exports.outputBigIntegerFormatter)(tx.gasPrice);
    }
    if (tx.maxFeePerGas) {
        modifiedTx.maxFeePerGas = (0, exports.outputBigIntegerFormatter)(tx.maxFeePerGas);
    }
    if (tx.maxPriorityFeePerGas) {
        modifiedTx.maxPriorityFeePerGas = (0, exports.outputBigIntegerFormatter)(tx.maxPriorityFeePerGas);
    }
    if (tx.type) {
        modifiedTx.type = (0, web3_utils_1.hexToNumber)(tx.type);
    }
    modifiedTx.value = (0, exports.outputBigIntegerFormatter)(tx.value);
    if (tx.to && (0, web3_utils_1.isAddress)(tx.to)) {
        // tx.to could be `0x0` or `null` while contract creation
        modifiedTx.to = (0, web3_utils_1.toChecksumAddress)(tx.to);
    }
    else {
        modifiedTx.to = undefined; // set to `null` if invalid address
    }
    if (tx.from) {
        modifiedTx.from = (0, web3_utils_1.toChecksumAddress)(tx.from);
    }
    return modifiedTx;
};
exports.outputTransactionFormatter = outputTransactionFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * @param topic
 */
// To align with specification we use the type "null" here
// eslint-disable-next-line @typescript-eslint/ban-types
const inputTopicFormatter = (topic) => {
    // Using "null" value intentionally for validation
    // eslint-disable-next-line no-null/no-null
    if ((0, web3_validator_1.isNullish)(topic))
        return null;
    const value = String(topic);
    return (0, web3_validator_1.isHex)(value) ? value : (0, web3_utils_1.fromUtf8)(value);
};
exports.inputTopicFormatter = inputTopicFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * @param filter
 */
const inputLogFormatter = (filter) => {
    var _a;
    const val = (0, web3_validator_1.isNullish)(filter)
        ? {}
        : (0, web3_utils_1.mergeDeep)({}, filter);
    // If options !== undefined, don't blow out existing data
    if ((0, web3_validator_1.isNullish)(val.fromBlock)) {
        val.fromBlock = web3_types_1.BlockTags.LATEST;
    }
    val.fromBlock = (0, exports.inputBlockNumberFormatter)(val.fromBlock);
    if (!(0, web3_validator_1.isNullish)(val.toBlock)) {
        val.toBlock = (0, exports.inputBlockNumberFormatter)(val.toBlock);
    }
    // make sure topics, get converted to hex
    val.topics = (_a = val.topics) !== null && _a !== void 0 ? _a : [];
    val.topics = val.topics.map(topic => Array.isArray(topic)
        ? topic.map(exports.inputTopicFormatter)
        : (0, exports.inputTopicFormatter)(topic));
    if (val.address) {
        val.address = Array.isArray(val.address)
            ? val.address.map(addr => (0, exports.inputAddressFormatter)(addr))
            : (0, exports.inputAddressFormatter)(val.address);
    }
    return val;
};
exports.inputLogFormatter = inputLogFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a log
 * @function outputLogFormatter
 */
const outputLogFormatter = (log) => {
    const modifiedLog = Object.assign({}, log);
    const logIndex = typeof log.logIndex === 'string'
        ? log.logIndex
        : (0, web3_utils_1.numberToHex)(log.logIndex);
    // generate a custom log id
    if (typeof log.blockHash === 'string' && typeof log.transactionHash === 'string') {
        const shaId = (0, web3_utils_1.sha3Raw)(`${log.blockHash.replace('0x', '')}${log.transactionHash.replace('0x', '')}${logIndex.replace('0x', '')}`);
        modifiedLog.id = `log_${shaId.replace('0x', '').slice(0, 8)}`;
    }
    else if (!log.id) {
        modifiedLog.id = undefined;
    }
    if (log.blockNumber && (0, web3_utils_1.isHexStrict)(log.blockNumber)) {
        modifiedLog.blockNumber = (0, web3_utils_1.hexToNumber)(log.blockNumber);
    }
    if (log.transactionIndex && (0, web3_utils_1.isHexStrict)(log.transactionIndex)) {
        modifiedLog.transactionIndex = (0, web3_utils_1.hexToNumber)(log.transactionIndex);
    }
    if (log.logIndex && (0, web3_utils_1.isHexStrict)(log.logIndex)) {
        modifiedLog.logIndex = (0, web3_utils_1.hexToNumber)(log.logIndex);
    }
    if (log.address) {
        modifiedLog.address = (0, web3_utils_1.toChecksumAddress)(log.address);
    }
    return modifiedLog;
};
exports.outputLogFormatter = outputLogFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a transaction receipt to its proper values
 */
const outputTransactionReceiptFormatter = (receipt) => {
    if (typeof receipt !== 'object') {
        throw new web3_errors_1.FormatterError(`Received receipt is invalid: ${String(receipt)}`);
    }
    const modifiedReceipt = Object.assign({}, receipt);
    if (receipt.blockNumber) {
        modifiedReceipt.blockNumber = (0, web3_utils_1.hexToNumber)(receipt.blockNumber);
    }
    if (receipt.transactionIndex) {
        modifiedReceipt.transactionIndex = (0, web3_utils_1.hexToNumber)(receipt.transactionIndex);
    }
    modifiedReceipt.cumulativeGasUsed = (0, web3_utils_1.hexToNumber)(receipt.cumulativeGasUsed);
    modifiedReceipt.gasUsed = (0, web3_utils_1.hexToNumber)(receipt.gasUsed);
    if (receipt.logs && Array.isArray(receipt.logs)) {
        modifiedReceipt.logs = receipt.logs.map(exports.outputLogFormatter);
    }
    if (receipt.effectiveGasPrice) {
        modifiedReceipt.effectiveGasPrice = (0, web3_utils_1.hexToNumber)(receipt.effectiveGasPrice);
    }
    if (receipt.contractAddress) {
        modifiedReceipt.contractAddress = (0, web3_utils_1.toChecksumAddress)(receipt.contractAddress);
    }
    if (receipt.status) {
        modifiedReceipt.status = Boolean(parseInt(receipt.status, 10));
    }
    return modifiedReceipt;
};
exports.outputTransactionReceiptFormatter = outputTransactionReceiptFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a block to its proper values
 * @function outputBlockFormatter
 */
const outputBlockFormatter = (block) => {
    const modifiedBlock = Object.assign({}, block);
    // transform to number
    modifiedBlock.gasLimit = (0, web3_utils_1.hexToNumber)(block.gasLimit);
    modifiedBlock.gasUsed = (0, web3_utils_1.hexToNumber)(block.gasUsed);
    modifiedBlock.size = (0, web3_utils_1.hexToNumber)(block.size);
    modifiedBlock.timestamp = (0, web3_utils_1.hexToNumber)(block.timestamp);
    if (block.number) {
        modifiedBlock.number = (0, web3_utils_1.hexToNumber)(block.number);
    }
    if (block.difficulty) {
        modifiedBlock.difficulty = (0, exports.outputBigIntegerFormatter)(block.difficulty);
    }
    if (block.totalDifficulty) {
        modifiedBlock.totalDifficulty = (0, exports.outputBigIntegerFormatter)(block.totalDifficulty);
    }
    if (block.transactions && Array.isArray(block.transactions)) {
        modifiedBlock.transactions = block.transactions.map(exports.outputTransactionFormatter);
    }
    if (block.miner) {
        modifiedBlock.miner = (0, web3_utils_1.toChecksumAddress)(block.miner);
    }
    if (block.baseFeePerGas) {
        modifiedBlock.baseFeePerGas = (0, exports.outputBigIntegerFormatter)(block.baseFeePerGas);
    }
    return modifiedBlock;
};
exports.outputBlockFormatter = outputBlockFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the input of a whisper post and converts all values to HEX
 */
const inputPostFormatter = (post) => {
    var _a;
    const modifiedPost = Object.assign({}, post);
    if (post.ttl) {
        modifiedPost.ttl = (0, web3_utils_1.numberToHex)(post.ttl);
    }
    if (post.workToProve) {
        modifiedPost.workToProve = (0, web3_utils_1.numberToHex)(post.workToProve);
    }
    if (post.priority) {
        modifiedPost.priority = (0, web3_utils_1.numberToHex)(post.priority);
    }
    // fallback
    if (post.topics && !Array.isArray(post.topics)) {
        modifiedPost.topics = post.topics ? [post.topics] : [];
    }
    // format the following options
    modifiedPost.topics = (_a = modifiedPost.topics) === null || _a === void 0 ? void 0 : _a.map(topic => topic.startsWith('0x') ? topic : (0, web3_utils_1.fromUtf8)(topic));
    return modifiedPost;
};
exports.inputPostFormatter = inputPostFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 * Formats the output of a received post message
 * @function outputPostFormatter
 */
const outputPostFormatter = (post) => {
    var _a;
    const modifiedPost = Object.assign({}, post);
    if (post.expiry) {
        modifiedPost.expiry = (0, web3_utils_1.hexToNumber)(post.expiry);
    }
    if (post.sent) {
        modifiedPost.sent = (0, web3_utils_1.hexToNumber)(post.sent);
    }
    if (post.ttl) {
        modifiedPost.ttl = (0, web3_utils_1.hexToNumber)(post.ttl);
    }
    if (post.workProved) {
        modifiedPost.workProved = (0, web3_utils_1.hexToNumber)(post.workProved);
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
    modifiedPost.topics = (_a = modifiedPost.topics) === null || _a === void 0 ? void 0 : _a.map(web3_utils_1.toUtf8);
    return modifiedPost;
};
exports.outputPostFormatter = outputPostFormatter;
/**
 * @deprecated Use format function from web3-utils package instead
 */
const outputSyncingFormatter = (result) => {
    const modifiedResult = Object.assign({}, result);
    modifiedResult.startingBlock = (0, web3_utils_1.hexToNumber)(result.startingBlock);
    modifiedResult.currentBlock = (0, web3_utils_1.hexToNumber)(result.currentBlock);
    modifiedResult.highestBlock = (0, web3_utils_1.hexToNumber)(result.highestBlock);
    if (result.knownStates) {
        modifiedResult.knownStates = (0, web3_utils_1.hexToNumber)(result.knownStates);
    }
    if (result.pulledStates) {
        modifiedResult.pulledStates = (0, web3_utils_1.hexToNumber)(result.pulledStates);
    }
    return modifiedResult;
};
exports.outputSyncingFormatter = outputSyncingFormatter;
//# sourceMappingURL=formatters.js.map