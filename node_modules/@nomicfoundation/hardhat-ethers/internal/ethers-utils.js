"use strict";
// these helpers functions were copied verbatim from ethers
Object.defineProperty(exports, "__esModule", { value: true });
exports.getRpcTransaction = exports.formatLog = exports.formatReceiptLog = exports.formatTransactionReceipt = exports.formatTransactionResponse = exports.formatBlock = exports.resolveProperties = exports.copyRequest = void 0;
const ethers_1 = require("ethers");
const errors_1 = require("./errors");
function copyRequest(req) {
    const result = {};
    // These could be addresses, ENS names or Addressables
    if (req.to !== null && req.to !== undefined) {
        result.to = req.to;
    }
    if (req.from !== null && req.from !== undefined) {
        result.from = req.from;
    }
    if (req.data !== null && req.data !== undefined) {
        result.data = (0, ethers_1.hexlify)(req.data);
    }
    const bigIntKeys = "chainId,gasLimit,gasPrice,maxFeePerGas,maxPriorityFeePerGas,value".split(/,/);
    for (const key of bigIntKeys) {
        if (!(key in req) ||
            req[key] === null ||
            req[key] === undefined) {
            continue;
        }
        result[key] = (0, ethers_1.getBigInt)(req[key], `request.${key}`);
    }
    const numberKeys = "type,nonce".split(/,/);
    for (const key of numberKeys) {
        if (!(key in req) ||
            req[key] === null ||
            req[key] === undefined) {
            continue;
        }
        result[key] = (0, ethers_1.getNumber)(req[key], `request.${key}`);
    }
    if (req.accessList !== null && req.accessList !== undefined) {
        result.accessList = (0, ethers_1.accessListify)(req.accessList);
    }
    if ("blockTag" in req) {
        result.blockTag = req.blockTag;
    }
    if ("enableCcipRead" in req) {
        result.enableCcipReadEnabled = Boolean(req.enableCcipRead);
    }
    if ("customData" in req) {
        result.customData = req.customData;
    }
    return result;
}
exports.copyRequest = copyRequest;
async function resolveProperties(value) {
    const keys = Object.keys(value);
    const results = await Promise.all(keys.map((k) => Promise.resolve(value[k])));
    return results.reduce((accum, v, index) => {
        accum[keys[index]] = v;
        return accum;
    }, {});
}
exports.resolveProperties = resolveProperties;
function formatBlock(value) {
    const result = _formatBlock(value);
    result.transactions = value.transactions.map((tx) => {
        if (typeof tx === "string") {
            return tx;
        }
        return formatTransactionResponse(tx);
    });
    return result;
}
exports.formatBlock = formatBlock;
const _formatBlock = object({
    hash: allowNull(formatHash),
    parentHash: formatHash,
    number: ethers_1.getNumber,
    timestamp: ethers_1.getNumber,
    nonce: allowNull(formatData),
    difficulty: ethers_1.getBigInt,
    gasLimit: ethers_1.getBigInt,
    gasUsed: ethers_1.getBigInt,
    miner: allowNull(ethers_1.getAddress),
    extraData: formatData,
    baseFeePerGas: allowNull(ethers_1.getBigInt),
});
function object(format, altNames) {
    return (value) => {
        const result = {};
        // eslint-disable-next-line guard-for-in
        for (const key in format) {
            let srcKey = key;
            if (altNames !== undefined && key in altNames && !(srcKey in value)) {
                for (const altKey of altNames[key]) {
                    if (altKey in value) {
                        srcKey = altKey;
                        break;
                    }
                }
            }
            try {
                const nv = format[key](value[srcKey]);
                if (nv !== undefined) {
                    result[key] = nv;
                }
            }
            catch (error) {
                const message = error instanceof Error ? error.message : "not-an-error";
                (0, ethers_1.assert)(false, `invalid value for value.${key} (${message})`, "BAD_DATA", { value });
            }
        }
        return result;
    };
}
function allowNull(format, nullValue) {
    return function (value) {
        // eslint-disable-next-line eqeqeq
        if (value === null || value === undefined) {
            return nullValue;
        }
        return format(value);
    };
}
function formatHash(value) {
    (0, ethers_1.assertArgument)((0, ethers_1.isHexString)(value, 32), "invalid hash", "value", value);
    return value;
}
function formatData(value) {
    (0, ethers_1.assertArgument)((0, ethers_1.isHexString)(value, true), "invalid data", "value", value);
    return value;
}
function formatTransactionResponse(value) {
    // Some clients (TestRPC) do strange things like return 0x0 for the
    // 0 address; correct this to be a real address
    // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
    if (value.to && (0, ethers_1.getBigInt)(value.to) === 0n) {
        value.to = "0x0000000000000000000000000000000000000000";
    }
    const result = object({
        hash: formatHash,
        type: (v) => {
            // eslint-disable-next-line eqeqeq
            if (v === "0x" || v == null) {
                return 0;
            }
            return (0, ethers_1.getNumber)(v);
        },
        accessList: allowNull(ethers_1.accessListify, null),
        blockHash: allowNull(formatHash, null),
        blockNumber: allowNull(ethers_1.getNumber, null),
        transactionIndex: allowNull(ethers_1.getNumber, null),
        from: ethers_1.getAddress,
        // either (gasPrice) or (maxPriorityFeePerGas + maxFeePerGas) must be set
        gasPrice: allowNull(ethers_1.getBigInt),
        maxPriorityFeePerGas: allowNull(ethers_1.getBigInt),
        maxFeePerGas: allowNull(ethers_1.getBigInt),
        gasLimit: ethers_1.getBigInt,
        to: allowNull(ethers_1.getAddress, null),
        value: ethers_1.getBigInt,
        nonce: ethers_1.getNumber,
        data: formatData,
        creates: allowNull(ethers_1.getAddress, null),
        chainId: allowNull(ethers_1.getBigInt, null),
    }, {
        data: ["input"],
        gasLimit: ["gas"],
    })(value);
    // If to and creates are empty, populate the creates from the value
    // eslint-disable-next-line eqeqeq
    if (result.to == null && result.creates == null) {
        result.creates = (0, ethers_1.getCreateAddress)(result);
    }
    // @TODO: Check fee data
    // Add an access list to supported transaction types
    // eslint-disable-next-line eqeqeq
    if ((value.type === 1 || value.type === 2) && value.accessList == null) {
        result.accessList = [];
    }
    // Compute the signature
    // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
    if (value.signature) {
        result.signature = ethers_1.Signature.from(value.signature);
    }
    else {
        result.signature = ethers_1.Signature.from(value);
    }
    // Some backends omit ChainId on legacy transactions, but we can compute it
    // eslint-disable-next-line eqeqeq
    if (result.chainId == null) {
        const chainId = result.signature.legacyChainId;
        // eslint-disable-next-line eqeqeq
        if (chainId != null) {
            result.chainId = chainId;
        }
    }
    // 0x0000... should actually be null
    // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
    if (result.blockHash && (0, ethers_1.getBigInt)(result.blockHash) === 0n) {
        result.blockHash = null;
    }
    return result;
}
exports.formatTransactionResponse = formatTransactionResponse;
function arrayOf(format) {
    return (array) => {
        if (!Array.isArray(array)) {
            throw new errors_1.HardhatEthersError("not an array");
        }
        return array.map((i) => format(i));
    };
}
const _formatReceiptLog = object({
    transactionIndex: ethers_1.getNumber,
    blockNumber: ethers_1.getNumber,
    transactionHash: formatHash,
    address: ethers_1.getAddress,
    topics: arrayOf(formatHash),
    data: formatData,
    index: ethers_1.getNumber,
    blockHash: formatHash,
}, {
    index: ["logIndex"],
});
const _formatTransactionReceipt = object({
    to: allowNull(ethers_1.getAddress, null),
    from: allowNull(ethers_1.getAddress, null),
    contractAddress: allowNull(ethers_1.getAddress, null),
    // should be allowNull(hash), but broken-EIP-658 support is handled in receipt
    index: ethers_1.getNumber,
    root: allowNull(ethers_1.hexlify),
    gasUsed: ethers_1.getBigInt,
    logsBloom: allowNull(formatData),
    blockHash: formatHash,
    hash: formatHash,
    logs: arrayOf(formatReceiptLog),
    blockNumber: ethers_1.getNumber,
    cumulativeGasUsed: ethers_1.getBigInt,
    effectiveGasPrice: allowNull(ethers_1.getBigInt),
    status: allowNull(ethers_1.getNumber),
    type: allowNull(ethers_1.getNumber, 0),
}, {
    effectiveGasPrice: ["gasPrice"],
    hash: ["transactionHash"],
    index: ["transactionIndex"],
});
function formatTransactionReceipt(value) {
    return _formatTransactionReceipt(value);
}
exports.formatTransactionReceipt = formatTransactionReceipt;
function formatReceiptLog(value) {
    return _formatReceiptLog(value);
}
exports.formatReceiptLog = formatReceiptLog;
function formatBoolean(value) {
    switch (value) {
        case true:
        case "true":
            return true;
        case false:
        case "false":
            return false;
    }
    (0, ethers_1.assertArgument)(false, `invalid boolean; ${JSON.stringify(value)}`, "value", value);
}
const _formatLog = object({
    address: ethers_1.getAddress,
    blockHash: formatHash,
    blockNumber: ethers_1.getNumber,
    data: formatData,
    index: ethers_1.getNumber,
    removed: formatBoolean,
    topics: arrayOf(formatHash),
    transactionHash: formatHash,
    transactionIndex: ethers_1.getNumber,
}, {
    index: ["logIndex"],
});
function formatLog(value) {
    return _formatLog(value);
}
exports.formatLog = formatLog;
function getRpcTransaction(tx) {
    const result = {};
    // JSON-RPC now requires numeric values to be "quantity" values
    [
        "chainId",
        "gasLimit",
        "gasPrice",
        "type",
        "maxFeePerGas",
        "maxPriorityFeePerGas",
        "nonce",
        "value",
    ].forEach((key) => {
        if (tx[key] === null || tx[key] === undefined) {
            return;
        }
        let dstKey = key;
        if (key === "gasLimit") {
            dstKey = "gas";
        }
        result[dstKey] = (0, ethers_1.toQuantity)((0, ethers_1.getBigInt)(tx[key], `tx.${key}`));
    });
    // Make sure addresses and data are lowercase
    ["from", "to", "data"].forEach((key) => {
        if (tx[key] === null || tx[key] === undefined) {
            return;
        }
        result[key] = (0, ethers_1.hexlify)(tx[key]);
    });
    // Normalize the access list object
    if (tx.accessList !== null && tx.accessList !== undefined) {
        result.accessList = (0, ethers_1.accessListify)(tx.accessList);
    }
    return result;
}
exports.getRpcTransaction = getRpcTransaction;
//# sourceMappingURL=ethers-utils.js.map