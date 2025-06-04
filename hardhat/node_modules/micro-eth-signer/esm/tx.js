import * as P from 'micro-packed';
import { addr } from "./address.js";
import { RLP } from "./rlp.js";
import { amounts, ethHex, isBytes, isObject } from "./utils.js";
// Transaction parsers
const _0n = BigInt(0);
const createTxMap = (versions) => {
    const ent = Object.entries(versions);
    // 'legacy' => {type, ver, coder}
    const typeMap = Object.fromEntries(ent.map(([type, coder], ver) => [type, { type, ver, coder }]));
    // '0' => {type, ver, coder}
    const verMap = Object.fromEntries(ent.map(([type, coder], ver) => [ver, { type, ver, coder }]));
    return P.wrap({
        encodeStream(w, value) {
            const t = value.type;
            if (!typeMap.hasOwnProperty(t))
                throw new Error(`txVersion: wrong type=${t}`);
            const curr = typeMap[t];
            if (t !== 'legacy')
                w.byte(curr.ver);
            curr.coder.encodeStream(w, value.data);
        },
        decodeStream(r) {
            const v = r.byte(true);
            if (v === 0xff)
                throw new Error('reserved version 0xff');
            // TODO: version=0 is legacy, but it is never wrapped in test vectors
            if (v === 0x00)
                throw new Error('version=0 unsupported');
            if (0 <= v && v <= 0x7f) {
                if (!verMap.hasOwnProperty(v.toString()))
                    throw new Error(`wrong version=${v}`);
                const curr = verMap[v];
                r.byte(false); // skip first byte
                const d = curr.coder.decodeStream(r);
                return { type: curr.type, data: d };
            }
            return { type: 'legacy', data: typeMap.legacy.coder.decodeStream(r) };
        },
    });
};
/**
 * Static struct could have been extracted into micro-packed, but we need a specific behavior:
 * - optional fields maybe either all present or all absent, enforced by type
 * - optional fields change the length of underlying array
 */
const isOptBig = (a) => a === undefined || typeof a === 'bigint';
const isNullOr0 = (a) => a === undefined || a === BigInt(0);
function assertYParityValid(elm) {
    // TODO: is this correct? elm = 0 default?
    if (elm === undefined)
        elm = 0;
    if (elm !== 0 && elm !== 1)
        throw new Error(`yParity wrong value=${elm} (${typeof elm})`);
}
// We don't know chainId when specific field coded yet.
const addrCoder = ethHex;
// Bytes32: VersionedHash, AccessListKey
function ensure32(b) {
    if (!isBytes(b) || b.length !== 32)
        throw new Error('expected 32 bytes');
    return b;
}
const Bytes32 = {
    encode: (from) => ethHex.encode(ensure32(from)),
    decode: (to) => ensure32(ethHex.decode(to)),
};
// Process v as (chainId, yParity) pair. Ethers.js-inspired logic:
//   - v=27/28 -> no chainId (pre eip155)
//   - r & s == 0 -> v = chainId
// Non-standard, but there is no other way to save chainId for unsignedTx.
// Case: unsigned tx for cold wallet for different chains, like mainnet & testnet.
//   - otherwise v = yParity + 2*chainId + 35
//   - allows to keep legacy logic here, instead of copying to Transaction
export const legacySig = {
    encode: (data) => {
        const { v, r, s } = data;
        if (v === undefined)
            return { chainId: undefined };
        // TODO: handle (invalid?) negative v
        if (typeof v !== 'bigint')
            throw new Error(`invalid v type=${typeof v}`);
        if ((r === undefined && s === undefined) || (r === _0n && s === _0n))
            return { chainId: v };
        if (v === BigInt(27))
            return { yParity: 0, chainId: undefined, r, s };
        if (v === BigInt(28))
            return { yParity: 1, chainId: undefined, r, s };
        if (v < BigInt(35))
            throw new Error(`wrong v=${v}`);
        const v2 = v - BigInt(35);
        return { chainId: v2 >> BigInt(1), yParity: Number(v2 & BigInt(1)), r, s };
    },
    decode: (data) => {
        aobj(data);
        const { chainId, yParity, r, s } = data;
        if (!isOptBig(chainId))
            throw new Error(`wrong chainId type=${typeof chainId}`);
        if (!isOptBig(r))
            throw new Error(`wrong r type=${typeof r}`);
        if (!isOptBig(s))
            throw new Error(`wrong s type=${typeof s}`);
        if (yParity !== undefined && typeof yParity !== 'number')
            throw new Error(`wrong yParity type=${typeof chainId}`);
        if (yParity === undefined) {
            if (chainId !== undefined) {
                if ((r !== undefined && r !== _0n) || (s !== undefined && s !== _0n))
                    throw new Error(`wrong unsigned legacy r=${r} s=${s}`);
                return { v: chainId, r: _0n, s: _0n };
            }
            // no parity, chainId, but r, s exists
            if ((r !== undefined && r !== _0n) || (s !== undefined && s !== _0n))
                throw new Error(`wrong unsigned legacy r=${r} s=${s}`);
            return {};
        }
        // parity exists, which means r & s should exist too!
        if (isNullOr0(r) || isNullOr0(s))
            throw new Error(`wrong unsigned legacy r=${r} s=${s}`);
        assertYParityValid(yParity);
        const v = chainId !== undefined
            ? BigInt(yParity) + (chainId * BigInt(2) + BigInt(35))
            : BigInt(yParity) + BigInt(27);
        return { v, r, s };
    },
};
const U64BE = P.coders.reverse(P.bigint(8, false, false, false));
const U256BE = P.coders.reverse(P.bigint(32, false, false, false));
// Small coder utils
// TODO: seems generic enought for packed? or RLP (seems useful for structured encoding/decoding of RLP stuff)
// Basic array coder
const array = (coder) => ({
    encode(from) {
        if (!Array.isArray(from))
            throw new Error('expected array');
        return from.map((i) => coder.encode(i));
    },
    decode(to) {
        if (!Array.isArray(to))
            throw new Error('expected array');
        return to.map((i) => coder.decode(i));
    },
});
// tuple -> struct
const struct = (fields) => ({
    encode(from) {
        if (!Array.isArray(from))
            throw new Error('expected array');
        const fNames = Object.keys(fields);
        if (from.length !== fNames.length)
            throw new Error('wrong array length');
        return Object.fromEntries(fNames.map((f, i) => [f, fields[f].encode(from[i])]));
    },
    decode(to) {
        const fNames = Object.keys(fields);
        if (!isObject(to))
            throw new Error('wrong struct object');
        return fNames.map((i) => fields[i].decode(to[i]));
    },
});
// U256BE in geth. But it is either 0 or 1. TODO: is this good enough?
const yParityCoder = P.coders.reverse(P.validate(P.int(1, false, false, false), (elm) => {
    assertYParityValid(elm);
    return elm;
}));
const accessListItem = struct({ address: addrCoder, storageKeys: array(Bytes32) });
export const authorizationRequest = struct({
    chainId: U256BE,
    address: addrCoder,
    nonce: U64BE,
});
// [chain_id, address, nonce, y_parity, r, s]
const authorizationItem = struct({
    chainId: U256BE,
    address: addrCoder,
    nonce: U64BE,
    yParity: yParityCoder,
    r: U256BE,
    s: U256BE,
});
/**
 * Field types, matching geth. Either u64 or u256.
 */
const coders = {
    chainId: U256BE, // Can fit into u64 (curr max is 0x57a238f93bf), but geth uses bigint
    nonce: U64BE,
    gasPrice: U256BE,
    maxPriorityFeePerGas: U256BE,
    maxFeePerGas: U256BE,
    gasLimit: U64BE,
    to: addrCoder,
    value: U256BE, // "Decimal" coder can be used, but it's harder to work with
    data: ethHex,
    accessList: array(accessListItem),
    maxFeePerBlobGas: U256BE,
    blobVersionedHashes: array(Bytes32),
    yParity: yParityCoder,
    v: U256BE,
    r: U256BE,
    s: U256BE,
    authorizationList: array(authorizationItem),
};
const signatureFields = new Set(['v', 'yParity', 'r', 's']);
// Mutates raw. Make sure to copy it in advance
export function removeSig(raw) {
    signatureFields.forEach((k) => {
        delete raw[k];
    });
    return raw;
}
/**
 * Defines RLP transaction with fields taken from `coders`.
 * @example
 *   const tx = txStruct(['nonce', 'gasPrice', 'value'] as const, ['v', 'r', 's'] as const)
 *   tx.nonce.decode(...);
 */
const txStruct = (reqf, optf) => {
    const allFields = reqf.concat(optf);
    // Check that all fields have known coders
    allFields.forEach((f) => {
        if (!coders.hasOwnProperty(f))
            throw new Error(`coder for field ${f} is not defined`);
    });
    const reqS = struct(Object.fromEntries(reqf.map((i) => [i, coders[i]])));
    const allS = struct(Object.fromEntries(allFields.map((i) => [i, coders[i]])));
    // e.g. eip1559 txs have valid lengths of 9 or 12 (unsigned / signed)
    const reql = reqf.length;
    const optl = reql + optf.length;
    const optFieldAt = (i) => reql + i;
    const isEmpty = (item) => item.length === 0;
    // TX is a bunch of fields in specific order. Field like nonce must always be at the same index.
    // We walk through all indexes in proper order.
    const fcoder = P.wrap({
        encodeStream(w, raw) {
            // If at least one optional key is present, we add whole optional block
            const hasOptional = optf.some((f) => raw.hasOwnProperty(f));
            const sCoder = hasOptional ? allS : reqS;
            RLP.encodeStream(w, sCoder.decode(raw));
        },
        decodeStream(r) {
            const decoded = RLP.decodeStream(r);
            if (!Array.isArray(decoded))
                throw new Error('txStruct: expected array from inner coder');
            const length = decoded.length;
            if (length !== reql && length !== optl)
                throw new Error(`txStruct: wrong inner length=${length}`);
            const sCoder = length === optl ? allS : reqS;
            if (length === optl && optf.every((_, i) => isEmpty(decoded[optFieldAt(i)])))
                throw new Error('all optional fields empty');
            // @ts-ignore TODO: fix type (there can be null in RLP)
            return sCoder.encode(decoded);
        },
    });
    fcoder.fields = reqf;
    fcoder.optionalFields = optf;
    fcoder.setOfAllFields = new Set(allFields.concat(['type']));
    return fcoder;
};
// prettier-ignore
const legacyInternal = txStruct([
    'nonce', 'gasPrice', 'gasLimit', 'to', 'value', 'data'
], ['v', 'r', 's']);
const legacy = (() => {
    const res = P.apply(legacyInternal, {
        decode: (data) => Object.assign({}, data, legacySig.decode(data)),
        encode: (data) => {
            const res = Object.assign({}, data);
            res.chainId = undefined;
            if (data.v) {
                const newV = legacySig.encode(data);
                removeSig(res);
                Object.assign(res, newV);
            }
            return res;
        },
    });
    res.fields = legacyInternal.fields.concat(['chainId']);
    // v, r, s -> yParity, r, s
    // TODO: what about chainId?
    res.optionalFields = ['yParity', 'r', 's'];
    res.setOfAllFields = new Set(res.fields.concat(res.optionalFields, ['type']));
    return res;
})();
// prettier-ignore
const eip2930 = txStruct([
    'chainId', 'nonce', 'gasPrice', 'gasLimit', 'to', 'value', 'data', 'accessList'
], ['yParity', 'r', 's']);
// prettier-ignore
const eip1559 = txStruct([
    'chainId', 'nonce', 'maxPriorityFeePerGas', 'maxFeePerGas', 'gasLimit', 'to', 'value', 'data', 'accessList'
], ['yParity', 'r', 's']);
// prettier-ignore
const eip4844 = txStruct([
    'chainId', 'nonce', 'maxPriorityFeePerGas', 'maxFeePerGas', 'gasLimit', 'to', 'value', 'data', 'accessList',
    'maxFeePerBlobGas', 'blobVersionedHashes'
], ['yParity', 'r', 's']);
// prettier-ignore
const eip7702 = txStruct([
    'chainId', 'nonce', 'maxPriorityFeePerGas', 'maxFeePerGas', 'gasLimit', 'to', 'value', 'data', 'accessList',
    'authorizationList'
], ['yParity', 'r', 's']);
export const TxVersions = {
    legacy, // 0x00 (kinda)
    eip2930, // 0x01
    eip1559, // 0x02
    eip4844, // 0x03
    eip7702, // 0x04
};
export const RawTx = P.apply(createTxMap(TxVersions), {
    // NOTE: we apply checksum to addresses here, since chainId is not available inside coders
    // By construction 'to' field is decoded before anything about chainId is known
    encode: (data) => {
        data.data.to = addr.addChecksum(data.data.to, true);
        if (data.type !== 'legacy' && data.data.accessList) {
            for (const item of data.data.accessList) {
                item.address = addr.addChecksum(item.address);
            }
        }
        if (data.type === 'eip7702' && data.data.authorizationList) {
            for (const item of data.data.authorizationList) {
                item.address = addr.addChecksum(item.address);
            }
        }
        return data;
    },
    // Nothing to check here, is validated in validator
    decode: (data) => data,
});
/**
 * Unchecked TX for debugging. Returns raw Uint8Array-s.
 * Handles versions - plain RLP will crash on it.
 */
export const RlpTx = createTxMap(Object.fromEntries(Object.keys(TxVersions).map((k) => [k, RLP])));
// prettier-ignore
// Basically all numbers. Can be useful if we decide to do converter from hex here
// const knownFieldsNoLeading0 = [
//   'nonce', 'maxPriorityFeePerGas', 'maxFeePerGas', 'gasLimit', 'value', 'yParity', 'r', 's'
// ] as const;
function abig(val) {
    if (typeof val !== 'bigint')
        throw new Error('value must be bigint');
}
function aobj(val) {
    if (typeof val !== 'object' || val == null)
        throw new Error('object expected');
}
function minmax(val, min, max, err) {
    if (!err)
        err = `>= ${min} and <= ${max}`;
    if (Number.isNaN(val) || val < min || val > max)
        throw new Error(`must be ${err}, not ${val}`);
}
// NOTE: non-strict validators can be removed (RawTx will handle that), but errors will be less user-friendly.
// On other hand, we twice per sig because tx is immutable
// data passed for composite checks (gasLimit * maxFeePerGas overflow and stuff) [not implemented yet]
const validators = {
    nonce(num, { strict }) {
        abig(num);
        if (strict)
            minmax(num, _0n, amounts.maxNonce);
        else
            minmax(BigInt(num), _0n, BigInt(Number.MAX_SAFE_INTEGER)); // amounts.maxUint64
    },
    maxFeePerGas(num, { strict }) {
        abig(num);
        if (strict)
            minmax(num, BigInt(1), amounts.maxGasPrice, '>= 1 wei and < 10000 gwei');
        else
            minmax(num, _0n, amounts.maxUint64);
    },
    maxPriorityFeePerGas(num, { strict, data }) {
        abig(num);
        if (strict)
            minmax(num, _0n, amounts.maxGasPrice, '>= 1 wei and < 10000 gwei');
        else
            minmax(num, _0n, amounts.maxUint64, '>= 1 wei and < 10000 gwei');
        if (strict && data && typeof data.maxFeePerGas === 'bigint' && data.maxFeePerGas < num) {
            throw new Error(`cannot be bigger than maxFeePerGas=${data.maxFeePerGas}`);
        }
    },
    gasLimit(num, { strict }) {
        abig(num);
        if (strict)
            minmax(num, amounts.minGasLimit, amounts.maxGasLimit);
        else
            minmax(num, _0n, amounts.maxUint64);
    },
    to(address, { strict, data }) {
        if (!addr.isValid(address, true))
            throw new Error('address checksum does not match');
        if (strict && address === '0x' && !data.data)
            throw new Error('Empty address (0x) without contract deployment code');
    },
    value(num) {
        abig(num);
        minmax(num, _0n, amounts.maxAmount, '>= 0 and < 100M eth');
    },
    data(val, { strict, data }) {
        if (typeof val !== 'string')
            throw new Error('data must be string');
        if (strict) {
            if (val.length > amounts.maxDataSize)
                throw new Error('data is too big: ' + val.length);
        }
        // NOTE: data is hex here
        if (data.to === '0x' && val.length > 2 * amounts.maxInitDataSize)
            throw new Error(`initcode is too big: ${val.length}`);
    },
    chainId(num, { strict, type }) {
        // chainId is optional for legacy transactions
        if (type === 'legacy' && num === undefined)
            return;
        abig(num);
        if (strict)
            minmax(num, BigInt(1), amounts.maxChainId, '>= 1 and <= 2**32-1');
    },
    accessList(list) {
        // NOTE: we cannot handle this validation in coder, since it requires chainId to calculate correct checksum
        for (const { address } of list) {
            if (!addr.isValid(address))
                throw new Error('address checksum does not match');
        }
    },
    authorizationList(list, opts) {
        for (const { address, nonce, chainId } of list) {
            if (!addr.isValid(address))
                throw new Error('address checksum does not match');
            // chainId in authorization list can be zero (==allow any chain)
            abig(chainId);
            if (opts.strict)
                minmax(chainId, _0n, amounts.maxChainId, '>= 0 and <= 2**32-1');
            this.nonce(nonce, opts);
        }
    },
};
export class AggregatedError extends Error {
    constructor(message, errors) {
        super();
        this.message = message;
        this.errors = errors;
    }
}
export function validateFields(type, data, strict = true, allowSignatureFields = true) {
    aobj(data);
    if (!TxVersions.hasOwnProperty(type))
        throw new Error(`unknown tx type=${type}`);
    const txType = TxVersions[type];
    const dataFields = new Set(Object.keys(data));
    const dataHas = (field) => dataFields.has(field);
    function checkField(field) {
        if (!dataHas(field))
            return { field, error: `field "${field}" must be present for tx type=${type}` };
        const val = data[field];
        try {
            if (validators.hasOwnProperty(field))
                validators[field](val, { data, strict, type });
            if (field === 'chainId')
                return; // chainId is validated, but can't be decoded
            coders[field].decode(val); // decoding may throw an error
        }
        catch (error) {
            // No early-return: when multiple fields have error, we should show them all.
            return { field, error: error.message };
        }
        return undefined;
    }
    // All fields are required.
    const reqErrs = txType.fields.map(checkField);
    // Signature fields should be all present or all missing
    const optErrs = txType.optionalFields.some(dataHas) ? txType.optionalFields.map(checkField) : [];
    // Check if user data has unexpected fields
    const unexpErrs = Object.keys(data).map((field) => {
        if (!txType.setOfAllFields.has(field))
            return { field, error: `unknown field "${field}" for tx type=${type}` };
        if (!allowSignatureFields && signatureFields.has(field))
            return {
                field,
                error: `field "${field}" is sig-related and must not be user-specified`,
            };
        return;
    });
    const errors = reqErrs
        .concat(optErrs, unexpErrs)
        .filter((val) => val !== undefined);
    if (errors.length > 0)
        throw new AggregatedError('fields had validation errors', errors);
}
// prettier-ignore
const sortedFieldOrder = [
    'to', 'value', 'nonce',
    'maxFeePerGas', 'maxFeePerBlobGas', 'maxPriorityFeePerGas', 'gasPrice', 'gasLimit',
    'accessList', 'authorizationList', 'blobVersionedHashes', 'chainId', 'data', 'type',
    'r', 's', 'yParity', 'v'
];
// TODO: remove any
export function sortRawData(raw) {
    const sortedRaw = {};
    sortedFieldOrder
        .filter((field) => raw.hasOwnProperty(field))
        .forEach((field) => {
        sortedRaw[field] = raw[field];
    });
    return sortedRaw;
}
export function decodeLegacyV(raw) {
    return legacySig.decode(raw).v;
}
// NOTE: for tests only, don't use
export const __tests = { legacySig, TxVersions };
//# sourceMappingURL=tx.js.map