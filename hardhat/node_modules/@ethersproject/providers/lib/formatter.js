"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.showThrottleMessage = exports.isCommunityResource = exports.isCommunityResourcable = exports.Formatter = void 0;
var address_1 = require("@ethersproject/address");
var bignumber_1 = require("@ethersproject/bignumber");
var bytes_1 = require("@ethersproject/bytes");
var constants_1 = require("@ethersproject/constants");
var properties_1 = require("@ethersproject/properties");
var transactions_1 = require("@ethersproject/transactions");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var Formatter = /** @class */ (function () {
    function Formatter() {
        this.formats = this.getDefaultFormats();
    }
    Formatter.prototype.getDefaultFormats = function () {
        var _this = this;
        var formats = ({});
        var address = this.address.bind(this);
        var bigNumber = this.bigNumber.bind(this);
        var blockTag = this.blockTag.bind(this);
        var data = this.data.bind(this);
        var hash = this.hash.bind(this);
        var hex = this.hex.bind(this);
        var number = this.number.bind(this);
        var type = this.type.bind(this);
        var strictData = function (v) { return _this.data(v, true); };
        formats.transaction = {
            hash: hash,
            type: type,
            accessList: Formatter.allowNull(this.accessList.bind(this), null),
            blockHash: Formatter.allowNull(hash, null),
            blockNumber: Formatter.allowNull(number, null),
            transactionIndex: Formatter.allowNull(number, null),
            confirmations: Formatter.allowNull(number, null),
            from: address,
            // either (gasPrice) or (maxPriorityFeePerGas + maxFeePerGas)
            // must be set
            gasPrice: Formatter.allowNull(bigNumber),
            maxPriorityFeePerGas: Formatter.allowNull(bigNumber),
            maxFeePerGas: Formatter.allowNull(bigNumber),
            gasLimit: bigNumber,
            to: Formatter.allowNull(address, null),
            value: bigNumber,
            nonce: number,
            data: data,
            r: Formatter.allowNull(this.uint256),
            s: Formatter.allowNull(this.uint256),
            v: Formatter.allowNull(number),
            creates: Formatter.allowNull(address, null),
            raw: Formatter.allowNull(data),
        };
        formats.transactionRequest = {
            from: Formatter.allowNull(address),
            nonce: Formatter.allowNull(number),
            gasLimit: Formatter.allowNull(bigNumber),
            gasPrice: Formatter.allowNull(bigNumber),
            maxPriorityFeePerGas: Formatter.allowNull(bigNumber),
            maxFeePerGas: Formatter.allowNull(bigNumber),
            to: Formatter.allowNull(address),
            value: Formatter.allowNull(bigNumber),
            data: Formatter.allowNull(strictData),
            type: Formatter.allowNull(number),
            accessList: Formatter.allowNull(this.accessList.bind(this), null),
        };
        formats.receiptLog = {
            transactionIndex: number,
            blockNumber: number,
            transactionHash: hash,
            address: address,
            topics: Formatter.arrayOf(hash),
            data: data,
            logIndex: number,
            blockHash: hash,
        };
        formats.receipt = {
            to: Formatter.allowNull(this.address, null),
            from: Formatter.allowNull(this.address, null),
            contractAddress: Formatter.allowNull(address, null),
            transactionIndex: number,
            // should be allowNull(hash), but broken-EIP-658 support is handled in receipt
            root: Formatter.allowNull(hex),
            gasUsed: bigNumber,
            logsBloom: Formatter.allowNull(data),
            blockHash: hash,
            transactionHash: hash,
            logs: Formatter.arrayOf(this.receiptLog.bind(this)),
            blockNumber: number,
            confirmations: Formatter.allowNull(number, null),
            cumulativeGasUsed: bigNumber,
            effectiveGasPrice: Formatter.allowNull(bigNumber),
            status: Formatter.allowNull(number),
            type: type
        };
        formats.block = {
            hash: Formatter.allowNull(hash),
            parentHash: hash,
            number: number,
            timestamp: number,
            nonce: Formatter.allowNull(hex),
            difficulty: this.difficulty.bind(this),
            gasLimit: bigNumber,
            gasUsed: bigNumber,
            miner: Formatter.allowNull(address),
            extraData: data,
            transactions: Formatter.allowNull(Formatter.arrayOf(hash)),
            baseFeePerGas: Formatter.allowNull(bigNumber)
        };
        formats.blockWithTransactions = (0, properties_1.shallowCopy)(formats.block);
        formats.blockWithTransactions.transactions = Formatter.allowNull(Formatter.arrayOf(this.transactionResponse.bind(this)));
        formats.filter = {
            fromBlock: Formatter.allowNull(blockTag, undefined),
            toBlock: Formatter.allowNull(blockTag, undefined),
            blockHash: Formatter.allowNull(hash, undefined),
            address: Formatter.allowNull(address, undefined),
            topics: Formatter.allowNull(this.topics.bind(this), undefined),
        };
        formats.filterLog = {
            blockNumber: Formatter.allowNull(number),
            blockHash: Formatter.allowNull(hash),
            transactionIndex: number,
            removed: Formatter.allowNull(this.boolean.bind(this)),
            address: address,
            data: Formatter.allowFalsish(data, "0x"),
            topics: Formatter.arrayOf(hash),
            transactionHash: hash,
            logIndex: number,
        };
        return formats;
    };
    Formatter.prototype.accessList = function (accessList) {
        return (0, transactions_1.accessListify)(accessList || []);
    };
    // Requires a BigNumberish that is within the IEEE754 safe integer range; returns a number
    // Strict! Used on input.
    Formatter.prototype.number = function (number) {
        if (number === "0x") {
            return 0;
        }
        return bignumber_1.BigNumber.from(number).toNumber();
    };
    Formatter.prototype.type = function (number) {
        if (number === "0x" || number == null) {
            return 0;
        }
        return bignumber_1.BigNumber.from(number).toNumber();
    };
    // Strict! Used on input.
    Formatter.prototype.bigNumber = function (value) {
        return bignumber_1.BigNumber.from(value);
    };
    // Requires a boolean, "true" or  "false"; returns a boolean
    Formatter.prototype.boolean = function (value) {
        if (typeof (value) === "boolean") {
            return value;
        }
        if (typeof (value) === "string") {
            value = value.toLowerCase();
            if (value === "true") {
                return true;
            }
            if (value === "false") {
                return false;
            }
        }
        throw new Error("invalid boolean - " + value);
    };
    Formatter.prototype.hex = function (value, strict) {
        if (typeof (value) === "string") {
            if (!strict && value.substring(0, 2) !== "0x") {
                value = "0x" + value;
            }
            if ((0, bytes_1.isHexString)(value)) {
                return value.toLowerCase();
            }
        }
        return logger.throwArgumentError("invalid hash", "value", value);
    };
    Formatter.prototype.data = function (value, strict) {
        var result = this.hex(value, strict);
        if ((result.length % 2) !== 0) {
            throw new Error("invalid data; odd-length - " + value);
        }
        return result;
    };
    // Requires an address
    // Strict! Used on input.
    Formatter.prototype.address = function (value) {
        return (0, address_1.getAddress)(value);
    };
    Formatter.prototype.callAddress = function (value) {
        if (!(0, bytes_1.isHexString)(value, 32)) {
            return null;
        }
        var address = (0, address_1.getAddress)((0, bytes_1.hexDataSlice)(value, 12));
        return (address === constants_1.AddressZero) ? null : address;
    };
    Formatter.prototype.contractAddress = function (value) {
        return (0, address_1.getContractAddress)(value);
    };
    // Strict! Used on input.
    Formatter.prototype.blockTag = function (blockTag) {
        if (blockTag == null) {
            return "latest";
        }
        if (blockTag === "earliest") {
            return "0x0";
        }
        switch (blockTag) {
            case "earliest": return "0x0";
            case "latest":
            case "pending":
            case "safe":
            case "finalized":
                return blockTag;
        }
        if (typeof (blockTag) === "number" || (0, bytes_1.isHexString)(blockTag)) {
            return (0, bytes_1.hexValue)(blockTag);
        }
        throw new Error("invalid blockTag");
    };
    // Requires a hash, optionally requires 0x prefix; returns prefixed lowercase hash.
    Formatter.prototype.hash = function (value, strict) {
        var result = this.hex(value, strict);
        if ((0, bytes_1.hexDataLength)(result) !== 32) {
            return logger.throwArgumentError("invalid hash", "value", value);
        }
        return result;
    };
    // Returns the difficulty as a number, or if too large (i.e. PoA network) null
    Formatter.prototype.difficulty = function (value) {
        if (value == null) {
            return null;
        }
        var v = bignumber_1.BigNumber.from(value);
        try {
            return v.toNumber();
        }
        catch (error) { }
        return null;
    };
    Formatter.prototype.uint256 = function (value) {
        if (!(0, bytes_1.isHexString)(value)) {
            throw new Error("invalid uint256");
        }
        return (0, bytes_1.hexZeroPad)(value, 32);
    };
    Formatter.prototype._block = function (value, format) {
        if (value.author != null && value.miner == null) {
            value.miner = value.author;
        }
        // The difficulty may need to come from _difficulty in recursed blocks
        var difficulty = (value._difficulty != null) ? value._difficulty : value.difficulty;
        var result = Formatter.check(format, value);
        result._difficulty = ((difficulty == null) ? null : bignumber_1.BigNumber.from(difficulty));
        return result;
    };
    Formatter.prototype.block = function (value) {
        return this._block(value, this.formats.block);
    };
    Formatter.prototype.blockWithTransactions = function (value) {
        return this._block(value, this.formats.blockWithTransactions);
    };
    // Strict! Used on input.
    Formatter.prototype.transactionRequest = function (value) {
        return Formatter.check(this.formats.transactionRequest, value);
    };
    Formatter.prototype.transactionResponse = function (transaction) {
        // Rename gas to gasLimit
        if (transaction.gas != null && transaction.gasLimit == null) {
            transaction.gasLimit = transaction.gas;
        }
        // Some clients (TestRPC) do strange things like return 0x0 for the
        // 0 address; correct this to be a real address
        if (transaction.to && bignumber_1.BigNumber.from(transaction.to).isZero()) {
            transaction.to = "0x0000000000000000000000000000000000000000";
        }
        // Rename input to data
        if (transaction.input != null && transaction.data == null) {
            transaction.data = transaction.input;
        }
        // If to and creates are empty, populate the creates from the transaction
        if (transaction.to == null && transaction.creates == null) {
            transaction.creates = this.contractAddress(transaction);
        }
        if ((transaction.type === 1 || transaction.type === 2) && transaction.accessList == null) {
            transaction.accessList = [];
        }
        var result = Formatter.check(this.formats.transaction, transaction);
        if (transaction.chainId != null) {
            var chainId = transaction.chainId;
            if ((0, bytes_1.isHexString)(chainId)) {
                chainId = bignumber_1.BigNumber.from(chainId).toNumber();
            }
            result.chainId = chainId;
        }
        else {
            var chainId = transaction.networkId;
            // geth-etc returns chainId
            if (chainId == null && result.v == null) {
                chainId = transaction.chainId;
            }
            if ((0, bytes_1.isHexString)(chainId)) {
                chainId = bignumber_1.BigNumber.from(chainId).toNumber();
            }
            if (typeof (chainId) !== "number" && result.v != null) {
                chainId = (result.v - 35) / 2;
                if (chainId < 0) {
                    chainId = 0;
                }
                chainId = parseInt(chainId);
            }
            if (typeof (chainId) !== "number") {
                chainId = 0;
            }
            result.chainId = chainId;
        }
        // 0x0000... should actually be null
        if (result.blockHash && result.blockHash.replace(/0/g, "") === "x") {
            result.blockHash = null;
        }
        return result;
    };
    Formatter.prototype.transaction = function (value) {
        return (0, transactions_1.parse)(value);
    };
    Formatter.prototype.receiptLog = function (value) {
        return Formatter.check(this.formats.receiptLog, value);
    };
    Formatter.prototype.receipt = function (value) {
        var result = Formatter.check(this.formats.receipt, value);
        // RSK incorrectly implemented EIP-658, so we munge things a bit here for it
        if (result.root != null) {
            if (result.root.length <= 4) {
                // Could be 0x00, 0x0, 0x01 or 0x1
                var value_1 = bignumber_1.BigNumber.from(result.root).toNumber();
                if (value_1 === 0 || value_1 === 1) {
                    // Make sure if both are specified, they match
                    if (result.status != null && (result.status !== value_1)) {
                        logger.throwArgumentError("alt-root-status/status mismatch", "value", { root: result.root, status: result.status });
                    }
                    result.status = value_1;
                    delete result.root;
                }
                else {
                    logger.throwArgumentError("invalid alt-root-status", "value.root", result.root);
                }
            }
            else if (result.root.length !== 66) {
                // Must be a valid bytes32
                logger.throwArgumentError("invalid root hash", "value.root", result.root);
            }
        }
        if (result.status != null) {
            result.byzantium = true;
        }
        return result;
    };
    Formatter.prototype.topics = function (value) {
        var _this = this;
        if (Array.isArray(value)) {
            return value.map(function (v) { return _this.topics(v); });
        }
        else if (value != null) {
            return this.hash(value, true);
        }
        return null;
    };
    Formatter.prototype.filter = function (value) {
        return Formatter.check(this.formats.filter, value);
    };
    Formatter.prototype.filterLog = function (value) {
        return Formatter.check(this.formats.filterLog, value);
    };
    Formatter.check = function (format, object) {
        var result = {};
        for (var key in format) {
            try {
                var value = format[key](object[key]);
                if (value !== undefined) {
                    result[key] = value;
                }
            }
            catch (error) {
                error.checkKey = key;
                error.checkValue = object[key];
                throw error;
            }
        }
        return result;
    };
    // if value is null-ish, nullValue is returned
    Formatter.allowNull = function (format, nullValue) {
        return (function (value) {
            if (value == null) {
                return nullValue;
            }
            return format(value);
        });
    };
    // If value is false-ish, replaceValue is returned
    Formatter.allowFalsish = function (format, replaceValue) {
        return (function (value) {
            if (!value) {
                return replaceValue;
            }
            return format(value);
        });
    };
    // Requires an Array satisfying check
    Formatter.arrayOf = function (format) {
        return (function (array) {
            if (!Array.isArray(array)) {
                throw new Error("not an array");
            }
            var result = [];
            array.forEach(function (value) {
                result.push(format(value));
            });
            return result;
        });
    };
    return Formatter;
}());
exports.Formatter = Formatter;
function isCommunityResourcable(value) {
    return (value && typeof (value.isCommunityResource) === "function");
}
exports.isCommunityResourcable = isCommunityResourcable;
function isCommunityResource(value) {
    return (isCommunityResourcable(value) && value.isCommunityResource());
}
exports.isCommunityResource = isCommunityResource;
// Show the throttle message only once
var throttleMessage = false;
function showThrottleMessage() {
    if (throttleMessage) {
        return;
    }
    throttleMessage = true;
    console.log("========= NOTICE =========");
    console.log("Request-Rate Exceeded  (this message will not be repeated)");
    console.log("");
    console.log("The default API keys for each service are provided as a highly-throttled,");
    console.log("community resource for low-traffic projects and early prototyping.");
    console.log("");
    console.log("While your application will continue to function, we highly recommended");
    console.log("signing up for your own API keys to improve performance, increase your");
    console.log("request rate/limit and enable other perks, such as metrics and advanced APIs.");
    console.log("");
    console.log("For more details: https:/\/docs.ethers.io/api-keys/");
    console.log("==========================");
}
exports.showThrottleMessage = showThrottleMessage;
//# sourceMappingURL=formatter.js.map