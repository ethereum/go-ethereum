"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    Object.defineProperty(o, k2, { enumerable: true, get: function() { return m[k]; } });
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.parse = exports.serialize = exports.accessListify = exports.recoverAddress = exports.computeAddress = exports.TransactionTypes = void 0;
var address_1 = require("@ethersproject/address");
var bignumber_1 = require("@ethersproject/bignumber");
var bytes_1 = require("@ethersproject/bytes");
var constants_1 = require("@ethersproject/constants");
var keccak256_1 = require("@ethersproject/keccak256");
var properties_1 = require("@ethersproject/properties");
var RLP = __importStar(require("@ethersproject/rlp"));
var signing_key_1 = require("@ethersproject/signing-key");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var TransactionTypes;
(function (TransactionTypes) {
    TransactionTypes[TransactionTypes["legacy"] = 0] = "legacy";
    TransactionTypes[TransactionTypes["eip2930"] = 1] = "eip2930";
    TransactionTypes[TransactionTypes["eip1559"] = 2] = "eip1559";
})(TransactionTypes = exports.TransactionTypes || (exports.TransactionTypes = {}));
;
///////////////////////////////
function handleAddress(value) {
    if (value === "0x") {
        return null;
    }
    return (0, address_1.getAddress)(value);
}
function handleNumber(value) {
    if (value === "0x") {
        return constants_1.Zero;
    }
    return bignumber_1.BigNumber.from(value);
}
// Legacy Transaction Fields
var transactionFields = [
    { name: "nonce", maxLength: 32, numeric: true },
    { name: "gasPrice", maxLength: 32, numeric: true },
    { name: "gasLimit", maxLength: 32, numeric: true },
    { name: "to", length: 20 },
    { name: "value", maxLength: 32, numeric: true },
    { name: "data" },
];
var allowedTransactionKeys = {
    chainId: true, data: true, gasLimit: true, gasPrice: true, nonce: true, to: true, type: true, value: true
};
function computeAddress(key) {
    var publicKey = (0, signing_key_1.computePublicKey)(key);
    return (0, address_1.getAddress)((0, bytes_1.hexDataSlice)((0, keccak256_1.keccak256)((0, bytes_1.hexDataSlice)(publicKey, 1)), 12));
}
exports.computeAddress = computeAddress;
function recoverAddress(digest, signature) {
    return computeAddress((0, signing_key_1.recoverPublicKey)((0, bytes_1.arrayify)(digest), signature));
}
exports.recoverAddress = recoverAddress;
function formatNumber(value, name) {
    var result = (0, bytes_1.stripZeros)(bignumber_1.BigNumber.from(value).toHexString());
    if (result.length > 32) {
        logger.throwArgumentError("invalid length for " + name, ("transaction:" + name), value);
    }
    return result;
}
function accessSetify(addr, storageKeys) {
    return {
        address: (0, address_1.getAddress)(addr),
        storageKeys: (storageKeys || []).map(function (storageKey, index) {
            if ((0, bytes_1.hexDataLength)(storageKey) !== 32) {
                logger.throwArgumentError("invalid access list storageKey", "accessList[" + addr + ":" + index + "]", storageKey);
            }
            return storageKey.toLowerCase();
        })
    };
}
function accessListify(value) {
    if (Array.isArray(value)) {
        return value.map(function (set, index) {
            if (Array.isArray(set)) {
                if (set.length > 2) {
                    logger.throwArgumentError("access list expected to be [ address, storageKeys[] ]", "value[" + index + "]", set);
                }
                return accessSetify(set[0], set[1]);
            }
            return accessSetify(set.address, set.storageKeys);
        });
    }
    var result = Object.keys(value).map(function (addr) {
        var storageKeys = value[addr].reduce(function (accum, storageKey) {
            accum[storageKey] = true;
            return accum;
        }, {});
        return accessSetify(addr, Object.keys(storageKeys).sort());
    });
    result.sort(function (a, b) { return (a.address.localeCompare(b.address)); });
    return result;
}
exports.accessListify = accessListify;
function formatAccessList(value) {
    return accessListify(value).map(function (set) { return [set.address, set.storageKeys]; });
}
function _serializeEip1559(transaction, signature) {
    // If there is an explicit gasPrice, make sure it matches the
    // EIP-1559 fees; otherwise they may not understand what they
    // think they are setting in terms of fee.
    if (transaction.gasPrice != null) {
        var gasPrice = bignumber_1.BigNumber.from(transaction.gasPrice);
        var maxFeePerGas = bignumber_1.BigNumber.from(transaction.maxFeePerGas || 0);
        if (!gasPrice.eq(maxFeePerGas)) {
            logger.throwArgumentError("mismatch EIP-1559 gasPrice != maxFeePerGas", "tx", {
                gasPrice: gasPrice,
                maxFeePerGas: maxFeePerGas
            });
        }
    }
    var fields = [
        formatNumber(transaction.chainId || 0, "chainId"),
        formatNumber(transaction.nonce || 0, "nonce"),
        formatNumber(transaction.maxPriorityFeePerGas || 0, "maxPriorityFeePerGas"),
        formatNumber(transaction.maxFeePerGas || 0, "maxFeePerGas"),
        formatNumber(transaction.gasLimit || 0, "gasLimit"),
        ((transaction.to != null) ? (0, address_1.getAddress)(transaction.to) : "0x"),
        formatNumber(transaction.value || 0, "value"),
        (transaction.data || "0x"),
        (formatAccessList(transaction.accessList || []))
    ];
    if (signature) {
        var sig = (0, bytes_1.splitSignature)(signature);
        fields.push(formatNumber(sig.recoveryParam, "recoveryParam"));
        fields.push((0, bytes_1.stripZeros)(sig.r));
        fields.push((0, bytes_1.stripZeros)(sig.s));
    }
    return (0, bytes_1.hexConcat)(["0x02", RLP.encode(fields)]);
}
function _serializeEip2930(transaction, signature) {
    var fields = [
        formatNumber(transaction.chainId || 0, "chainId"),
        formatNumber(transaction.nonce || 0, "nonce"),
        formatNumber(transaction.gasPrice || 0, "gasPrice"),
        formatNumber(transaction.gasLimit || 0, "gasLimit"),
        ((transaction.to != null) ? (0, address_1.getAddress)(transaction.to) : "0x"),
        formatNumber(transaction.value || 0, "value"),
        (transaction.data || "0x"),
        (formatAccessList(transaction.accessList || []))
    ];
    if (signature) {
        var sig = (0, bytes_1.splitSignature)(signature);
        fields.push(formatNumber(sig.recoveryParam, "recoveryParam"));
        fields.push((0, bytes_1.stripZeros)(sig.r));
        fields.push((0, bytes_1.stripZeros)(sig.s));
    }
    return (0, bytes_1.hexConcat)(["0x01", RLP.encode(fields)]);
}
// Legacy Transactions and EIP-155
function _serialize(transaction, signature) {
    (0, properties_1.checkProperties)(transaction, allowedTransactionKeys);
    var raw = [];
    transactionFields.forEach(function (fieldInfo) {
        var value = transaction[fieldInfo.name] || ([]);
        var options = {};
        if (fieldInfo.numeric) {
            options.hexPad = "left";
        }
        value = (0, bytes_1.arrayify)((0, bytes_1.hexlify)(value, options));
        // Fixed-width field
        if (fieldInfo.length && value.length !== fieldInfo.length && value.length > 0) {
            logger.throwArgumentError("invalid length for " + fieldInfo.name, ("transaction:" + fieldInfo.name), value);
        }
        // Variable-width (with a maximum)
        if (fieldInfo.maxLength) {
            value = (0, bytes_1.stripZeros)(value);
            if (value.length > fieldInfo.maxLength) {
                logger.throwArgumentError("invalid length for " + fieldInfo.name, ("transaction:" + fieldInfo.name), value);
            }
        }
        raw.push((0, bytes_1.hexlify)(value));
    });
    var chainId = 0;
    if (transaction.chainId != null) {
        // A chainId was provided; if non-zero we'll use EIP-155
        chainId = transaction.chainId;
        if (typeof (chainId) !== "number") {
            logger.throwArgumentError("invalid transaction.chainId", "transaction", transaction);
        }
    }
    else if (signature && !(0, bytes_1.isBytesLike)(signature) && signature.v > 28) {
        // No chainId provided, but the signature is signing with EIP-155; derive chainId
        chainId = Math.floor((signature.v - 35) / 2);
    }
    // We have an EIP-155 transaction (chainId was specified and non-zero)
    if (chainId !== 0) {
        raw.push((0, bytes_1.hexlify)(chainId)); // @TODO: hexValue?
        raw.push("0x");
        raw.push("0x");
    }
    // Requesting an unsigned transaction
    if (!signature) {
        return RLP.encode(raw);
    }
    // The splitSignature will ensure the transaction has a recoveryParam in the
    // case that the signTransaction function only adds a v.
    var sig = (0, bytes_1.splitSignature)(signature);
    // We pushed a chainId and null r, s on for hashing only; remove those
    var v = 27 + sig.recoveryParam;
    if (chainId !== 0) {
        raw.pop();
        raw.pop();
        raw.pop();
        v += chainId * 2 + 8;
        // If an EIP-155 v (directly or indirectly; maybe _vs) was provided, check it!
        if (sig.v > 28 && sig.v !== v) {
            logger.throwArgumentError("transaction.chainId/signature.v mismatch", "signature", signature);
        }
    }
    else if (sig.v !== v) {
        logger.throwArgumentError("transaction.chainId/signature.v mismatch", "signature", signature);
    }
    raw.push((0, bytes_1.hexlify)(v));
    raw.push((0, bytes_1.stripZeros)((0, bytes_1.arrayify)(sig.r)));
    raw.push((0, bytes_1.stripZeros)((0, bytes_1.arrayify)(sig.s)));
    return RLP.encode(raw);
}
function serialize(transaction, signature) {
    // Legacy and EIP-155 Transactions
    if (transaction.type == null || transaction.type === 0) {
        if (transaction.accessList != null) {
            logger.throwArgumentError("untyped transactions do not support accessList; include type: 1", "transaction", transaction);
        }
        return _serialize(transaction, signature);
    }
    // Typed Transactions (EIP-2718)
    switch (transaction.type) {
        case 1:
            return _serializeEip2930(transaction, signature);
        case 2:
            return _serializeEip1559(transaction, signature);
        default:
            break;
    }
    return logger.throwError("unsupported transaction type: " + transaction.type, logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
        operation: "serializeTransaction",
        transactionType: transaction.type
    });
}
exports.serialize = serialize;
function _parseEipSignature(tx, fields, serialize) {
    try {
        var recid = handleNumber(fields[0]).toNumber();
        if (recid !== 0 && recid !== 1) {
            throw new Error("bad recid");
        }
        tx.v = recid;
    }
    catch (error) {
        logger.throwArgumentError("invalid v for transaction type: 1", "v", fields[0]);
    }
    tx.r = (0, bytes_1.hexZeroPad)(fields[1], 32);
    tx.s = (0, bytes_1.hexZeroPad)(fields[2], 32);
    try {
        var digest = (0, keccak256_1.keccak256)(serialize(tx));
        tx.from = recoverAddress(digest, { r: tx.r, s: tx.s, recoveryParam: tx.v });
    }
    catch (error) { }
}
function _parseEip1559(payload) {
    var transaction = RLP.decode(payload.slice(1));
    if (transaction.length !== 9 && transaction.length !== 12) {
        logger.throwArgumentError("invalid component count for transaction type: 2", "payload", (0, bytes_1.hexlify)(payload));
    }
    var maxPriorityFeePerGas = handleNumber(transaction[2]);
    var maxFeePerGas = handleNumber(transaction[3]);
    var tx = {
        type: 2,
        chainId: handleNumber(transaction[0]).toNumber(),
        nonce: handleNumber(transaction[1]).toNumber(),
        maxPriorityFeePerGas: maxPriorityFeePerGas,
        maxFeePerGas: maxFeePerGas,
        gasPrice: null,
        gasLimit: handleNumber(transaction[4]),
        to: handleAddress(transaction[5]),
        value: handleNumber(transaction[6]),
        data: transaction[7],
        accessList: accessListify(transaction[8]),
    };
    // Unsigned EIP-1559 Transaction
    if (transaction.length === 9) {
        return tx;
    }
    tx.hash = (0, keccak256_1.keccak256)(payload);
    _parseEipSignature(tx, transaction.slice(9), _serializeEip1559);
    return tx;
}
function _parseEip2930(payload) {
    var transaction = RLP.decode(payload.slice(1));
    if (transaction.length !== 8 && transaction.length !== 11) {
        logger.throwArgumentError("invalid component count for transaction type: 1", "payload", (0, bytes_1.hexlify)(payload));
    }
    var tx = {
        type: 1,
        chainId: handleNumber(transaction[0]).toNumber(),
        nonce: handleNumber(transaction[1]).toNumber(),
        gasPrice: handleNumber(transaction[2]),
        gasLimit: handleNumber(transaction[3]),
        to: handleAddress(transaction[4]),
        value: handleNumber(transaction[5]),
        data: transaction[6],
        accessList: accessListify(transaction[7])
    };
    // Unsigned EIP-2930 Transaction
    if (transaction.length === 8) {
        return tx;
    }
    tx.hash = (0, keccak256_1.keccak256)(payload);
    _parseEipSignature(tx, transaction.slice(8), _serializeEip2930);
    return tx;
}
// Legacy Transactions and EIP-155
function _parse(rawTransaction) {
    var transaction = RLP.decode(rawTransaction);
    if (transaction.length !== 9 && transaction.length !== 6) {
        logger.throwArgumentError("invalid raw transaction", "rawTransaction", rawTransaction);
    }
    var tx = {
        nonce: handleNumber(transaction[0]).toNumber(),
        gasPrice: handleNumber(transaction[1]),
        gasLimit: handleNumber(transaction[2]),
        to: handleAddress(transaction[3]),
        value: handleNumber(transaction[4]),
        data: transaction[5],
        chainId: 0
    };
    // Legacy unsigned transaction
    if (transaction.length === 6) {
        return tx;
    }
    try {
        tx.v = bignumber_1.BigNumber.from(transaction[6]).toNumber();
    }
    catch (error) {
        // @TODO: What makes snese to do? The v is too big
        return tx;
    }
    tx.r = (0, bytes_1.hexZeroPad)(transaction[7], 32);
    tx.s = (0, bytes_1.hexZeroPad)(transaction[8], 32);
    if (bignumber_1.BigNumber.from(tx.r).isZero() && bignumber_1.BigNumber.from(tx.s).isZero()) {
        // EIP-155 unsigned transaction
        tx.chainId = tx.v;
        tx.v = 0;
    }
    else {
        // Signed Transaction
        tx.chainId = Math.floor((tx.v - 35) / 2);
        if (tx.chainId < 0) {
            tx.chainId = 0;
        }
        var recoveryParam = tx.v - 27;
        var raw = transaction.slice(0, 6);
        if (tx.chainId !== 0) {
            raw.push((0, bytes_1.hexlify)(tx.chainId));
            raw.push("0x");
            raw.push("0x");
            recoveryParam -= tx.chainId * 2 + 8;
        }
        var digest = (0, keccak256_1.keccak256)(RLP.encode(raw));
        try {
            tx.from = recoverAddress(digest, { r: (0, bytes_1.hexlify)(tx.r), s: (0, bytes_1.hexlify)(tx.s), recoveryParam: recoveryParam });
        }
        catch (error) { }
        tx.hash = (0, keccak256_1.keccak256)(rawTransaction);
    }
    tx.type = null;
    return tx;
}
function parse(rawTransaction) {
    var payload = (0, bytes_1.arrayify)(rawTransaction);
    // Legacy and EIP-155 Transactions
    if (payload[0] > 0x7f) {
        return _parse(payload);
    }
    // Typed Transaction (EIP-2718)
    switch (payload[0]) {
        case 1:
            return _parseEip2930(payload);
        case 2:
            return _parseEip1559(payload);
        default:
            break;
    }
    return logger.throwError("unsupported transaction type: " + payload[0], logger_1.Logger.errors.UNSUPPORTED_OPERATION, {
        operation: "parseTransaction",
        transactionType: payload[0]
    });
}
exports.parse = parse;
//# sourceMappingURL=index.js.map