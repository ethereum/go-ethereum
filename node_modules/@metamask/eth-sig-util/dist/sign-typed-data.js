"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.recoverTypedSignature = exports.signTypedData = exports.typedSignatureHash = exports.TypedDataUtils = exports.TYPED_MESSAGE_SCHEMA = exports.SignTypedDataVersion = void 0;
const ethereumjs_util_1 = require("ethereumjs-util");
const ethereumjs_abi_1 = require("ethereumjs-abi");
const utils_1 = require("./utils");
/**
 * Represents the version of `signTypedData` being used.
 *
 * V1 is based upon [an early version of EIP-712](https://github.com/ethereum/EIPs/pull/712/commits/21abe254fe0452d8583d5b132b1d7be87c0439ca)
 * that lacked some later security improvements, and should generally be neglected in favor of
 * later versions.
 *
 * V3 is based on EIP-712, except that arrays and recursive data structures are not supported.
 *
 * V4 is based on EIP-712, and includes full support of arrays and recursive data structures.
 */
var SignTypedDataVersion;
(function (SignTypedDataVersion) {
    SignTypedDataVersion["V1"] = "V1";
    SignTypedDataVersion["V3"] = "V3";
    SignTypedDataVersion["V4"] = "V4";
})(SignTypedDataVersion = exports.SignTypedDataVersion || (exports.SignTypedDataVersion = {}));
exports.TYPED_MESSAGE_SCHEMA = {
    type: 'object',
    properties: {
        types: {
            type: 'object',
            additionalProperties: {
                type: 'array',
                items: {
                    type: 'object',
                    properties: {
                        name: { type: 'string' },
                        type: { type: 'string' },
                    },
                    required: ['name', 'type'],
                },
            },
        },
        primaryType: { type: 'string' },
        domain: { type: 'object' },
        message: { type: 'object' },
    },
    required: ['types', 'primaryType', 'domain', 'message'],
};
/**
 * Validate that the given value is a valid version string.
 *
 * @param version - The version value to validate.
 * @param allowedVersions - A list of allowed versions. If omitted, all versions are assumed to be
 * allowed.
 */
function validateVersion(version, allowedVersions) {
    if (!Object.keys(SignTypedDataVersion).includes(version)) {
        throw new Error(`Invalid version: '${version}'`);
    }
    else if (allowedVersions && !allowedVersions.includes(version)) {
        throw new Error(`SignTypedDataVersion not allowed: '${version}'. Allowed versions are: ${allowedVersions.join(', ')}`);
    }
}
/**
 * Encode a single field.
 *
 * @param types - All type definitions.
 * @param name - The name of the field to encode.
 * @param type - The type of the field being encoded.
 * @param value - The value to encode.
 * @param version - The EIP-712 version the encoding should comply with.
 * @returns Encoded representation of the field.
 */
function encodeField(types, name, type, value, version) {
    validateVersion(version, [SignTypedDataVersion.V3, SignTypedDataVersion.V4]);
    if (types[type] !== undefined) {
        return [
            'bytes32',
            version === SignTypedDataVersion.V4 && value == null // eslint-disable-line no-eq-null
                ? '0x0000000000000000000000000000000000000000000000000000000000000000'
                : ethereumjs_util_1.keccak(encodeData(type, value, types, version)),
        ];
    }
    if (value === undefined) {
        throw new Error(`missing value for field ${name} of type ${type}`);
    }
    if (type === 'bytes') {
        return ['bytes32', ethereumjs_util_1.keccak(value)];
    }
    if (type === 'string') {
        // convert string to buffer - prevents ethUtil from interpreting strings like '0xabcd' as hex
        if (typeof value === 'string') {
            value = Buffer.from(value, 'utf8');
        }
        return ['bytes32', ethereumjs_util_1.keccak(value)];
    }
    if (type.lastIndexOf(']') === type.length - 1) {
        if (version === SignTypedDataVersion.V3) {
            throw new Error('Arrays are unimplemented in encodeData; use V4 extension');
        }
        const parsedType = type.slice(0, type.lastIndexOf('['));
        const typeValuePairs = value.map((item) => encodeField(types, name, parsedType, item, version));
        return [
            'bytes32',
            ethereumjs_util_1.keccak(ethereumjs_abi_1.rawEncode(typeValuePairs.map(([t]) => t), typeValuePairs.map(([, v]) => v))),
        ];
    }
    return [type, value];
}
/**
 * Encodes an object by encoding and concatenating each of its members.
 *
 * @param primaryType - The root type.
 * @param data - The object to encode.
 * @param types - Type definitions for all types included in the message.
 * @param version - The EIP-712 version the encoding should comply with.
 * @returns An encoded representation of an object.
 */
function encodeData(primaryType, data, types, version) {
    validateVersion(version, [SignTypedDataVersion.V3, SignTypedDataVersion.V4]);
    const encodedTypes = ['bytes32'];
    const encodedValues = [hashType(primaryType, types)];
    for (const field of types[primaryType]) {
        if (version === SignTypedDataVersion.V3 && data[field.name] === undefined) {
            continue;
        }
        const [type, value] = encodeField(types, field.name, field.type, data[field.name], version);
        encodedTypes.push(type);
        encodedValues.push(value);
    }
    return ethereumjs_abi_1.rawEncode(encodedTypes, encodedValues);
}
/**
 * Encodes the type of an object by encoding a comma delimited list of its members.
 *
 * @param primaryType - The root type to encode.
 * @param types - Type definitions for all types included in the message.
 * @returns An encoded representation of the primary type.
 */
function encodeType(primaryType, types) {
    let result = '';
    const unsortedDeps = findTypeDependencies(primaryType, types);
    unsortedDeps.delete(primaryType);
    const deps = [primaryType, ...Array.from(unsortedDeps).sort()];
    for (const type of deps) {
        const children = types[type];
        if (!children) {
            throw new Error(`No type definition specified: ${type}`);
        }
        result += `${type}(${types[type]
            .map(({ name, type: t }) => `${t} ${name}`)
            .join(',')})`;
    }
    return result;
}
/**
 * Finds all types within a type definition object.
 *
 * @param primaryType - The root type.
 * @param types - Type definitions for all types included in the message.
 * @param results - The current set of accumulated types.
 * @returns The set of all types found in the type definition.
 */
function findTypeDependencies(primaryType, types, results = new Set()) {
    [primaryType] = primaryType.match(/^\w*/u);
    if (results.has(primaryType) || types[primaryType] === undefined) {
        return results;
    }
    results.add(primaryType);
    for (const field of types[primaryType]) {
        findTypeDependencies(field.type, types, results);
    }
    return results;
}
/**
 * Hashes an object.
 *
 * @param primaryType - The root type.
 * @param data - The object to hash.
 * @param types - Type definitions for all types included in the message.
 * @param version - The EIP-712 version the encoding should comply with.
 * @returns The hash of the object.
 */
function hashStruct(primaryType, data, types, version) {
    validateVersion(version, [SignTypedDataVersion.V3, SignTypedDataVersion.V4]);
    return ethereumjs_util_1.keccak(encodeData(primaryType, data, types, version));
}
/**
 * Hashes the type of an object.
 *
 * @param primaryType - The root type to hash.
 * @param types - Type definitions for all types included in the message.
 * @returns The hash of the object type.
 */
function hashType(primaryType, types) {
    return ethereumjs_util_1.keccak(encodeType(primaryType, types));
}
/**
 * Removes properties from a message object that are not defined per EIP-712.
 *
 * @param data - The typed message object.
 * @returns The typed message object with only allowed fields.
 */
function sanitizeData(data) {
    const sanitizedData = {};
    for (const key in exports.TYPED_MESSAGE_SCHEMA.properties) {
        if (data[key]) {
            sanitizedData[key] = data[key];
        }
    }
    if ('types' in sanitizedData) {
        sanitizedData.types = Object.assign({ EIP712Domain: [] }, sanitizedData.types);
    }
    return sanitizedData;
}
/**
 * Hash a typed message according to EIP-712. The returned message starts with the EIP-712 prefix,
 * which is "1901", followed by the hash of the domain separator, then the data (if any).
 * The result is hashed again and returned.
 *
 * This function does not sign the message. The resulting hash must still be signed to create an
 * EIP-712 signature.
 *
 * @param typedData - The typed message to hash.
 * @param version - The EIP-712 version the encoding should comply with.
 * @returns The hash of the typed message.
 */
function eip712Hash(typedData, version) {
    validateVersion(version, [SignTypedDataVersion.V3, SignTypedDataVersion.V4]);
    const sanitizedData = sanitizeData(typedData);
    const parts = [Buffer.from('1901', 'hex')];
    parts.push(hashStruct('EIP712Domain', sanitizedData.domain, sanitizedData.types, version));
    if (sanitizedData.primaryType !== 'EIP712Domain') {
        parts.push(hashStruct(
        // TODO: Validate that this is a string, so this type cast can be removed.
        sanitizedData.primaryType, sanitizedData.message, sanitizedData.types, version));
    }
    return ethereumjs_util_1.keccak(Buffer.concat(parts));
}
/**
 * A collection of utility functions used for signing typed data.
 */
exports.TypedDataUtils = {
    encodeData,
    encodeType,
    findTypeDependencies,
    hashStruct,
    hashType,
    sanitizeData,
    eip712Hash,
};
/**
 * Generate the "V1" hash for the provided typed message.
 *
 * The hash will be generated in accordance with an earlier version of the EIP-712
 * specification. This hash is used in `signTypedData_v1`.
 *
 * @param typedData - The typed message.
 * @returns The '0x'-prefixed hex encoded hash representing the type of the provided message.
 */
function typedSignatureHash(typedData) {
    const hashBuffer = _typedSignatureHash(typedData);
    return ethereumjs_util_1.bufferToHex(hashBuffer);
}
exports.typedSignatureHash = typedSignatureHash;
/**
 * Generate the "V1" hash for the provided typed message.
 *
 * The hash will be generated in accordance with an earlier version of the EIP-712
 * specification. This hash is used in `signTypedData_v1`.
 *
 * @param typedData - The typed message.
 * @returns The hash representing the type of the provided message.
 */
function _typedSignatureHash(typedData) {
    const error = new Error('Expect argument to be non-empty array');
    if (typeof typedData !== 'object' ||
        !('length' in typedData) ||
        !typedData.length) {
        throw error;
    }
    const data = typedData.map(function (e) {
        if (e.type !== 'bytes') {
            return e.value;
        }
        return utils_1.legacyToBuffer(e.value);
    });
    const types = typedData.map(function (e) {
        return e.type;
    });
    const schema = typedData.map(function (e) {
        if (!e.name) {
            throw error;
        }
        return `${e.type} ${e.name}`;
    });
    return ethereumjs_abi_1.soliditySHA3(['bytes32', 'bytes32'], [
        ethereumjs_abi_1.soliditySHA3(new Array(typedData.length).fill('string'), schema),
        ethereumjs_abi_1.soliditySHA3(types, data),
    ]);
}
/**
 * Sign typed data according to EIP-712. The signing differs based upon the `version`.
 *
 * V1 is based upon [an early version of EIP-712](https://github.com/ethereum/EIPs/pull/712/commits/21abe254fe0452d8583d5b132b1d7be87c0439ca)
 * that lacked some later security improvements, and should generally be neglected in favor of
 * later versions.
 *
 * V3 is based on [EIP-712](https://eips.ethereum.org/EIPS/eip-712), except that arrays and
 * recursive data structures are not supported.
 *
 * V4 is based on [EIP-712](https://eips.ethereum.org/EIPS/eip-712), and includes full support of
 * arrays and recursive data structures.
 *
 * @param options - The signing options.
 * @param options.privateKey - The private key to sign with.
 * @param options.data - The typed data to sign.
 * @param options.version - The signing version to use.
 * @returns The '0x'-prefixed hex encoded signature.
 */
function signTypedData({ privateKey, data, version, }) {
    validateVersion(version);
    if (utils_1.isNullish(data)) {
        throw new Error('Missing data parameter');
    }
    else if (utils_1.isNullish(privateKey)) {
        throw new Error('Missing private key parameter');
    }
    const messageHash = version === SignTypedDataVersion.V1
        ? _typedSignatureHash(data)
        : exports.TypedDataUtils.eip712Hash(data, version);
    const sig = ethereumjs_util_1.ecsign(messageHash, privateKey);
    return utils_1.concatSig(ethereumjs_util_1.toBuffer(sig.v), sig.r, sig.s);
}
exports.signTypedData = signTypedData;
/**
 * Recover the address of the account that created the given EIP-712
 * signature. The version provided must match the version used to
 * create the signature.
 *
 * @param options - The signature recovery options.
 * @param options.data - The typed data that was signed.
 * @param options.signature - The '0x-prefixed hex encoded message signature.
 * @param options.version - The signing version to use.
 * @returns The '0x'-prefixed hex address of the signer.
 */
function recoverTypedSignature({ data, signature, version, }) {
    validateVersion(version);
    if (utils_1.isNullish(data)) {
        throw new Error('Missing data parameter');
    }
    else if (utils_1.isNullish(signature)) {
        throw new Error('Missing signature parameter');
    }
    const messageHash = version === SignTypedDataVersion.V1
        ? _typedSignatureHash(data)
        : exports.TypedDataUtils.eip712Hash(data, version);
    const publicKey = utils_1.recoverPublicKey(messageHash, signature);
    const sender = ethereumjs_util_1.publicToAddress(publicKey);
    return ethereumjs_util_1.bufferToHex(sender);
}
exports.recoverTypedSignature = recoverTypedSignature;
//# sourceMappingURL=sign-typed-data.js.map