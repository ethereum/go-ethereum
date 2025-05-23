"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.extractStructNameIfAvailable = exports.parseEvmType = exports.StructName = void 0;
const logger_1 = require("../utils/logger");
const normalizeName_1 = require("./normalizeName");
class StructName {
    constructor(_identifier, _namespace) {
        this.identifier = (0, normalizeName_1.normalizeName)(_identifier);
        if (_namespace)
            this.namespace = (0, normalizeName_1.normalizeName)(_namespace);
    }
    toString() {
        if (this.namespace) {
            return `${this.namespace}.${this.identifier}`;
        }
        return this.identifier;
    }
    merge(other) {
        return new StructName(other.identifier || this.identifier, other.namespace || this.namespace);
    }
}
exports.StructName = StructName;
const isUIntTypeRegex = /^uint([0-9]*)$/;
const isIntTypeRegex = /^int([0-9]*)$/;
const isBytesTypeRegex = /^bytes([0-9]+)$/;
function parseEvmType(rawType, components, internalType) {
    const lastChar = rawType[rawType.length - 1];
    // first we parse array type
    if (lastChar === ']') {
        let finishArrayTypeIndex = rawType.length - 2;
        while (rawType[finishArrayTypeIndex] !== '[') {
            finishArrayTypeIndex--;
        }
        const arraySizeRaw = rawType.slice(finishArrayTypeIndex + 1, rawType.length - 1);
        const arraySize = arraySizeRaw !== '' ? parseInt(arraySizeRaw) : undefined;
        const restOfTheType = rawType.slice(0, finishArrayTypeIndex);
        const result = {
            type: 'array',
            itemType: parseEvmType(restOfTheType, components, internalType),
            originalType: rawType,
        };
        if (arraySize)
            result.size = arraySize;
        const structName = extractStructNameIfAvailable(internalType);
        if (structName)
            result.structName = structName;
        return result;
    }
    // otherwise this has to be primitive type
    // deal with simple to parse types
    switch (rawType) {
        case 'bool':
            return { type: 'boolean', originalType: rawType };
        case 'address':
            return { type: 'address', originalType: rawType };
        case 'string':
            return { type: 'string', originalType: rawType };
        case 'byte':
            return { type: 'bytes', size: 1, originalType: rawType };
        case 'bytes':
            return { type: 'dynamic-bytes', originalType: rawType };
        case 'tuple':
            if (!components)
                throw new Error('Tuple specified without components!');
            const result = { type: 'tuple', components, originalType: rawType };
            const structName = extractStructNameIfAvailable(internalType);
            if (structName)
                result.structName = structName;
            return result;
    }
    if (isUIntTypeRegex.test(rawType)) {
        const match = isUIntTypeRegex.exec(rawType);
        return { type: 'uinteger', bits: parseInt(match[1] || '256'), originalType: rawType };
    }
    if (isIntTypeRegex.test(rawType)) {
        const match = isIntTypeRegex.exec(rawType);
        return { type: 'integer', bits: parseInt(match[1] || '256'), originalType: rawType };
    }
    if (isBytesTypeRegex.test(rawType)) {
        const match = isBytesTypeRegex.exec(rawType);
        return { type: 'bytes', size: parseInt(match[1] || '1'), originalType: rawType };
    }
    if (internalType === null || internalType === void 0 ? void 0 : internalType.startsWith('enum')) {
        return parseEvmType('uint8'); // this is a best effort approach. Sometimes enums can be uint16 too. Read more: https://github.com/ethereum-ts/TypeChain/pull/281#discussion_r513303099
    }
    if (internalType === null || internalType === void 0 ? void 0 : internalType.startsWith('contract')) {
        return { type: 'address', originalType: rawType };
    }
    // unknown type
    logger_1.logger.warn(`Could not parse type: ${rawType} with internal type: ${internalType}.\n\nPlease submit a GitHub Issue to the TypeChain team with the failing contract/library.`);
    return { type: 'unknown', originalType: rawType };
}
exports.parseEvmType = parseEvmType;
/** @internal */
function extractStructNameIfAvailable(internalType) {
    var _a;
    if (internalType === null || internalType === void 0 ? void 0 : internalType.startsWith('struct ')) {
        // get rid of "struct " in the beginning
        let nameStr = internalType.slice(7);
        // get rid of all array signs at the end
        const arrayMarker = (_a = nameStr.match(/((?:\[\d*\])+)$/)) === null || _a === void 0 ? void 0 : _a[1];
        if (arrayMarker) {
            nameStr = nameStr.slice(0, nameStr.length - arrayMarker.length);
        }
        if (nameStr.indexOf('.') !== -1) {
            const [namespace, identifier] = nameStr.split('.');
            return new StructName(identifier, namespace);
        }
        return new StructName(nameStr);
    }
}
exports.extractStructNameIfAvailable = extractStructNameIfAvailable;
//# sourceMappingURL=parseEvmType.js.map