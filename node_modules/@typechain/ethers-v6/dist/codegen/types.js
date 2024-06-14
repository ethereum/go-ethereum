"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.generateOutputComplexTypesAsObject = exports.generateOutputComplexTypeAsTuple = exports.generateOutputComplexType = exports.generateInputComplexTypeAsTuple = exports.generateObjectTypeLiteral = exports.generateOutputType = exports.generateInputType = exports.generateOutputTypes = exports.generateInputTypes = void 0;
/* eslint-disable import/no-extraneous-dependencies */
const lodash_1 = require("lodash");
const common_1 = require("../common");
const reserved_keywords_1 = require("./reserved-keywords");
function generateInputTypes(input, options) {
    if (input.length === 0) {
        return '';
    }
    return (input
        .map((input, index) => `${input.name || `arg${index}`}: ${generateInputType(options, input.type)}`)
        .join(', ') + ', ');
}
exports.generateInputTypes = generateInputTypes;
function generateOutputTypes(options, outputs) {
    if (!options.returnResultObject && outputs.length === 1) {
        return generateOutputType(options, outputs[0].type);
    }
    else {
        return generateOutputComplexType(outputs, options);
    }
}
exports.generateOutputTypes = generateOutputTypes;
// https://docs.ethers.io/ethers.js/html/api-contract.html#types
function generateInputType(options, evmType) {
    switch (evmType.type) {
        case 'integer':
            return 'BigNumberish';
        case 'uinteger':
            return 'BigNumberish';
        case 'address':
            return 'AddressLike';
        case 'bytes':
        case 'dynamic-bytes':
            return 'BytesLike';
        case 'array':
            return generateArrayOrTupleType(generateInputType(options, evmType.itemType), evmType.size);
        case 'boolean':
            return 'boolean';
        case 'string':
            return 'string';
        case 'tuple':
            if (evmType.structName && options.useStructs) {
                return evmType.structName.toString() + common_1.STRUCT_INPUT_POSTFIX;
            }
            return generateObjectTypeLiteral(evmType, generateInputType.bind(null, { ...options, useStructs: true }));
        case 'unknown':
            return 'any';
    }
}
exports.generateInputType = generateInputType;
function generateOutputType(options, evmType) {
    switch (evmType.type) {
        case 'integer':
        case 'uinteger':
            return 'bigint';
        case 'address':
            return 'string';
        case 'void':
            return 'void';
        case 'bytes':
        case 'dynamic-bytes':
            return 'string';
        case 'array':
            return generateArrayOrTupleType(generateOutputType(options, evmType.itemType), evmType.size);
        case 'boolean':
            return 'boolean';
        case 'string':
            return 'string';
        case 'tuple':
            if (evmType.structName && options.useStructs) {
                return evmType.structName.toString() + common_1.STRUCT_OUTPUT_POSTFIX;
            }
            return generateOutputComplexType(evmType.components, { ...options, useStructs: true });
        case 'unknown':
            return 'any';
    }
}
exports.generateOutputType = generateOutputType;
function generateObjectTypeLiteral(tuple, generator) {
    return '{' + tuple.components.map((component) => `${component.name}: ${generator(component.type)}`).join(', ') + '}';
}
exports.generateObjectTypeLiteral = generateObjectTypeLiteral;
function generateInputComplexTypeAsTuple(components, options) {
    return `[${components
        .map((t) => (options.includeLabelsInTupleTypes && !!t.name
        ? `${t.name}${reserved_keywords_1.reservedKeywordsLabels.has(t.name) ? '_' : ''}: `
        : '') + generateInputType(options, t.type))
        .join(', ')}]`;
}
exports.generateInputComplexTypeAsTuple = generateInputComplexTypeAsTuple;
/**
 * Always return an array type; if there are named outputs, merge them to that type
 * this generates slightly better typings fixing: https://github.com/ethereum-ts/TypeChain/issues/232
 **/
function generateOutputComplexType(components, options) {
    const existingOutputComponents = (0, lodash_1.compact)([
        generateOutputComplexTypeAsTuple(components, options),
        generateOutputComplexTypesAsObject(components, options),
    ]);
    return existingOutputComponents.join(' & ');
}
exports.generateOutputComplexType = generateOutputComplexType;
function generateOutputComplexTypeAsTuple(components, options) {
    return `[${components
        .map((t) => (options.includeLabelsInTupleTypes && !!t.name
        ? `${t.name}${reserved_keywords_1.reservedKeywordsLabels.has(t.name) ? '_' : ''}: `
        : '') + generateOutputType(options, t.type))
        .join(', ')}]`;
}
exports.generateOutputComplexTypeAsTuple = generateOutputComplexTypeAsTuple;
function generateOutputComplexTypesAsObject(components, options) {
    let namedElementsCode;
    const namedElements = components.filter((e) => !!e.name);
    if (namedElements.length > 0) {
        namedElementsCode =
            '{' + namedElements.map((t) => `${t.name}: ${generateOutputType(options, t.type)}`).join(', ') + ' }';
    }
    return namedElementsCode;
}
exports.generateOutputComplexTypesAsObject = generateOutputComplexTypesAsObject;
function generateArrayOrTupleType(item, length) {
    if (length !== undefined && length < 6) {
        return `[${Array(length).fill(item).join(', ')}]`;
    }
    else {
        return `${item}[]`;
    }
}
//# sourceMappingURL=types.js.map