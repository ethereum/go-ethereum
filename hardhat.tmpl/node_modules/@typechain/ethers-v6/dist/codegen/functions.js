"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.FUNCTION_IMPORTS = exports.generateParamNames = exports.generateDecodeFunctionResultOverload = exports.generateEncodeFunctionDataOverload = exports.generateGetFunctionForContract = exports.generateGetFunctionForInterface = exports.generateFunctionNameOrSignature = exports.generateInterfaceFunctionDescription = exports.codegenForOverloadedFunctions = exports.codegenFunctions = void 0;
/* eslint-disable import/no-extraneous-dependencies */
const typechain_1 = require("typechain");
const types_1 = require("./types");
function codegenFunctions(options, fns) {
    if (fns.length === 1) {
        if (options.codegenConfig.alwaysGenerateOverloads) {
            return generateFunction(fns[0]) + codegenForOverloadedFunctions(fns);
        }
        else {
            return generateFunction(fns[0]);
        }
    }
    return codegenForOverloadedFunctions(fns);
}
exports.codegenFunctions = codegenFunctions;
function codegenForOverloadedFunctions(fns) {
    return fns.map((fn) => generateFunction(fn, `"${(0, typechain_1.getSignatureForFn)(fn)}"`)).join('\n');
}
exports.codegenForOverloadedFunctions = codegenForOverloadedFunctions;
function isPayable(fn) {
    return fn.stateMutability === 'payable';
}
function generateFunctionReturnType(fn) {
    let stateMutability;
    if ((0, typechain_1.isConstant)(fn) || (0, typechain_1.isConstantFn)(fn)) {
        stateMutability = 'view';
    }
    else if (isPayable(fn)) {
        stateMutability = 'payable';
    }
    else {
        stateMutability = 'nonpayable';
    }
    return `TypedContractMethod<
      [${(0, types_1.generateInputTypes)(fn.inputs, { useStructs: true })}],
      [${(0, types_1.generateOutputTypes)({ returnResultObject: false, useStructs: true }, fn.outputs)}],
      '${stateMutability}'
    >`;
}
function generateFunction(fn, overloadedName) {
    return `
    ${generateFunctionDocumentation(fn.documentation)}
    ${overloadedName !== null && overloadedName !== void 0 ? overloadedName : fn.name}: ${generateFunctionReturnType(fn)}
    `;
}
function generateFunctionDocumentation(doc) {
    if (!doc)
        return '';
    let docString = '/**';
    if (doc.details)
        docString += `\n * ${doc.details}`;
    if (doc.notice)
        docString += `\n * ${doc.notice}`;
    const params = Object.entries(doc.params || {});
    if (params.length) {
        params.forEach(([key, value]) => {
            docString += `\n * @param ${key} ${value}`;
        });
    }
    if (doc.return)
        docString += `\n * @returns ${doc.return}`;
    docString += '\n */';
    return docString;
}
function generateInterfaceFunctionDescription(fn) {
    return `'${(0, typechain_1.getSignatureForFn)(fn)}': FunctionFragment;`;
}
exports.generateInterfaceFunctionDescription = generateInterfaceFunctionDescription;
function generateFunctionNameOrSignature(fn, useSignature) {
    return useSignature ? (0, typechain_1.getSignatureForFn)(fn) : fn.name;
}
exports.generateFunctionNameOrSignature = generateFunctionNameOrSignature;
function generateGetFunctionForInterface(args) {
    if (args.length === 0)
        return '';
    return `getFunction(nameOrSignature: ${args.map((s) => `"${s}"`).join(' | ')}): FunctionFragment;`;
}
exports.generateGetFunctionForInterface = generateGetFunctionForInterface;
function generateGetFunctionForContract(fn, useSignature) {
    return `getFunction(nameOrSignature: '${generateFunctionNameOrSignature(fn, useSignature)}'): ${generateFunctionReturnType(fn)};`;
}
exports.generateGetFunctionForContract = generateGetFunctionForContract;
function generateEncodeFunctionDataOverload(fn, useSignature) {
    const methodInputs = [`functionFragment: '${useSignature ? (0, typechain_1.getSignatureForFn)(fn) : fn.name}'`];
    if (fn.inputs.length) {
        methodInputs.push(`values: [${fn.inputs.map((input) => (0, types_1.generateInputType)({ useStructs: true }, input.type)).join(', ')}]`);
    }
    else {
        methodInputs.push('values?: undefined');
    }
    return `encodeFunctionData(${methodInputs.join(', ')}): string;`;
}
exports.generateEncodeFunctionDataOverload = generateEncodeFunctionDataOverload;
function generateDecodeFunctionResultOverload(fn, useSignature) {
    return `decodeFunctionResult(functionFragment: '${useSignature ? (0, typechain_1.getSignatureForFn)(fn) : fn.name}', data: BytesLike): Result;`;
}
exports.generateDecodeFunctionResultOverload = generateDecodeFunctionResultOverload;
function generateParamNames(params) {
    return params.map((param, index) => (param.name ? (0, typechain_1.createPositionalIdentifier)(param.name) : `arg${index}`)).join(', ');
}
exports.generateParamNames = generateParamNames;
exports.FUNCTION_IMPORTS = ['TypedContractMethod', 'NonPayableOverrides', 'PayableOverrides', 'ViewOverrides'];
//# sourceMappingURL=functions.js.map