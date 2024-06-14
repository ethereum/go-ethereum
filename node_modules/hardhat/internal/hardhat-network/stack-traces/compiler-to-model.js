"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.createModelsAndDecodeBytecodes = void 0;
const debug_1 = __importDefault(require("debug"));
const library_utils_1 = require("./library-utils");
const model_1 = require("./model");
const source_maps_1 = require("./source-maps");
const abi = require("ethereumjs-abi");
const log = (0, debug_1.default)("hardhat:core:hardhat-network:compiler-to-model");
function createModelsAndDecodeBytecodes(solcVersion, compilerInput, compilerOutput) {
    const fileIdToSourceFile = new Map();
    const contractIdToContract = new Map();
    createSourcesModelFromAst(compilerOutput, compilerInput, fileIdToSourceFile, contractIdToContract);
    const bytecodes = decodeBytecodes(solcVersion, compilerOutput, fileIdToSourceFile, contractIdToContract);
    correctSelectors(bytecodes, compilerOutput);
    return bytecodes;
}
exports.createModelsAndDecodeBytecodes = createModelsAndDecodeBytecodes;
function createSourcesModelFromAst(compilerOutput, compilerInput, fileIdToSourceFile, contractIdToContract) {
    const contractIdToLinearizedBaseContractIds = new Map();
    // Create a `sourceName => contract => abi` mapping
    const sourceNameToContractToAbi = new Map();
    for (const [sourceName, contracts] of Object.entries(compilerOutput.contracts)) {
        const contractToAbi = new Map();
        sourceNameToContractToAbi.set(sourceName, contractToAbi);
        for (const [contractName, contract] of Object.entries(contracts)) {
            contractToAbi.set(contractName, contract.abi);
        }
    }
    for (const [sourceName, source] of Object.entries(compilerOutput.sources)) {
        const contractToAbi = sourceNameToContractToAbi.get(sourceName);
        const file = new model_1.SourceFile(sourceName, compilerInput.sources[sourceName].content);
        fileIdToSourceFile.set(source.id, file);
        for (const node of source.ast.nodes) {
            if (node.nodeType === "ContractDefinition") {
                const contractType = contractKindToContractType(node.contractKind);
                if (contractType === undefined) {
                    continue;
                }
                const contractAbi = contractToAbi?.get(node.name);
                processContractAstNode(file, node, fileIdToSourceFile, contractType, contractIdToContract, contractIdToLinearizedBaseContractIds, contractAbi);
            }
            // top-level functions
            if (node.nodeType === "FunctionDefinition") {
                processFunctionDefinitionAstNode(node, fileIdToSourceFile, undefined, file);
            }
        }
    }
    applyContractsInheritance(contractIdToContract, contractIdToLinearizedBaseContractIds);
}
function processContractAstNode(file, contractNode, fileIdToSourceFile, contractType, contractIdToContract, contractIdToLinearizedBaseContractIds, contractAbi) {
    const contractLocation = astSrcToSourceLocation(contractNode.src, fileIdToSourceFile);
    const contract = new model_1.Contract(contractNode.name, contractType, contractLocation);
    contractIdToContract.set(contractNode.id, contract);
    contractIdToLinearizedBaseContractIds.set(contractNode.id, contractNode.linearizedBaseContracts);
    file.addContract(contract);
    for (const node of contractNode.nodes) {
        if (node.nodeType === "FunctionDefinition") {
            const functionAbis = contractAbi?.filter((abiEntry) => abiEntry.name === node.name);
            processFunctionDefinitionAstNode(node, fileIdToSourceFile, contract, file, functionAbis);
        }
        else if (node.nodeType === "ModifierDefinition") {
            processModifierDefinitionAstNode(node, fileIdToSourceFile, contract, file);
        }
        else if (node.nodeType === "VariableDeclaration") {
            const getterAbi = contractAbi?.find((abiEntry) => abiEntry.name === node.name);
            processVariableDeclarationAstNode(node, fileIdToSourceFile, contract, file, getterAbi);
        }
    }
}
function processFunctionDefinitionAstNode(functionDefinitionNode, fileIdToSourceFile, contract, file, functionAbis) {
    if (functionDefinitionNode.implemented === false) {
        return;
    }
    const functionType = functionDefinitionKindToFunctionType(functionDefinitionNode.kind);
    const functionLocation = astSrcToSourceLocation(functionDefinitionNode.src, fileIdToSourceFile);
    const visibility = astVisibilityToVisibility(functionDefinitionNode.visibility);
    let selector;
    if (functionType === model_1.ContractFunctionType.FUNCTION &&
        (visibility === model_1.ContractFunctionVisibility.EXTERNAL ||
            visibility === model_1.ContractFunctionVisibility.PUBLIC)) {
        selector = astFunctionDefinitionToSelector(functionDefinitionNode);
    }
    // function can be overloaded, match the abi by the selector
    const matchingFunctionAbi = functionAbis?.find((functionAbi) => {
        if (functionAbi.name === undefined) {
            return false;
        }
        const functionAbiSelector = abi.methodID(functionAbi.name, functionAbi.inputs?.map((input) => input.type) ?? []);
        if (selector === undefined || functionAbiSelector === undefined) {
            return false;
        }
        return selector.equals(functionAbiSelector);
    });
    const paramTypes = matchingFunctionAbi?.inputs?.map((input) => input.type);
    const cf = new model_1.ContractFunction(functionDefinitionNode.name, functionType, functionLocation, contract, visibility, functionDefinitionNode.stateMutability === "payable", selector, paramTypes);
    if (contract !== undefined) {
        contract.addLocalFunction(cf);
    }
    file.addFunction(cf);
}
function processModifierDefinitionAstNode(modifierDefinitionNode, fileIdToSourceFile, contract, file) {
    const functionLocation = astSrcToSourceLocation(modifierDefinitionNode.src, fileIdToSourceFile);
    const cf = new model_1.ContractFunction(modifierDefinitionNode.name, model_1.ContractFunctionType.MODIFIER, functionLocation, contract);
    contract.addLocalFunction(cf);
    file.addFunction(cf);
}
function canonicalAbiTypeForElementaryOrUserDefinedTypes(keyType) {
    if (isElementaryType(keyType)) {
        return toCanonicalAbiType(keyType.name);
    }
    if (isEnumType(keyType)) {
        return "uint256";
    }
    if (isContractType(keyType)) {
        return "address";
    }
    return undefined;
}
function getPublicVariableSelectorFromDeclarationAstNode(variableDeclaration) {
    if (variableDeclaration.functionSelector !== undefined) {
        return Buffer.from(variableDeclaration.functionSelector, "hex");
    }
    const paramTypes = [];
    // VariableDeclaration nodes for function parameters or state variables will always
    // have their typeName fields defined.
    let nextType = variableDeclaration.typeName;
    while (true) {
        if (nextType.nodeType === "Mapping") {
            const canonicalType = canonicalAbiTypeForElementaryOrUserDefinedTypes(nextType.keyType);
            paramTypes.push(canonicalType);
            nextType = nextType.valueType;
        }
        else {
            if (nextType.nodeType === "ArrayTypeName") {
                paramTypes.push("uint256");
            }
            break;
        }
    }
    return abi.methodID(variableDeclaration.name, paramTypes);
}
function processVariableDeclarationAstNode(variableDeclarationNode, fileIdToSourceFile, contract, file, getterAbi) {
    const visibility = astVisibilityToVisibility(variableDeclarationNode.visibility);
    // Variables can't be external
    if (visibility !== model_1.ContractFunctionVisibility.PUBLIC) {
        return;
    }
    const functionLocation = astSrcToSourceLocation(variableDeclarationNode.src, fileIdToSourceFile);
    const paramTypes = getterAbi?.inputs?.map((input) => input.type);
    const cf = new model_1.ContractFunction(variableDeclarationNode.name, model_1.ContractFunctionType.GETTER, functionLocation, contract, visibility, false, // Getters aren't payable
    getPublicVariableSelectorFromDeclarationAstNode(variableDeclarationNode), paramTypes);
    contract.addLocalFunction(cf);
    file.addFunction(cf);
}
function applyContractsInheritance(contractIdToContract, contractIdToLinearizedBaseContractIds) {
    for (const [cid, contract] of contractIdToContract.entries()) {
        const inheritanceIds = contractIdToLinearizedBaseContractIds.get(cid);
        for (const baseId of inheritanceIds) {
            const baseContract = contractIdToContract.get(baseId);
            if (baseContract === undefined) {
                // This list includes interface, which we don't model
                continue;
            }
            contract.addNextLinearizedBaseContract(baseContract);
        }
    }
}
function decodeBytecodes(solcVersion, compilerOutput, fileIdToSourceFile, contractIdToContract) {
    const bytecodes = [];
    for (const contract of contractIdToContract.values()) {
        const contractFile = contract.location.file.sourceName;
        const contractEvmOutput = compilerOutput.contracts[contractFile][contract.name].evm;
        const contractAbiOutput = compilerOutput.contracts[contractFile][contract.name].abi;
        for (const abiItem of contractAbiOutput) {
            if (abiItem.type === "error") {
                const customError = model_1.CustomError.fromABI(abiItem.name, abiItem.inputs);
                if (customError !== undefined) {
                    contract.addCustomError(customError);
                }
                else {
                    log(`Couldn't build CustomError for error '${abiItem.name}'`);
                }
            }
        }
        // This is an abstract contract
        if (contractEvmOutput.bytecode.object === "") {
            continue;
        }
        const deploymentBytecode = decodeEvmBytecode(contract, solcVersion, true, contractEvmOutput.bytecode, fileIdToSourceFile);
        const runtimeBytecode = decodeEvmBytecode(contract, solcVersion, false, contractEvmOutput.deployedBytecode, fileIdToSourceFile);
        bytecodes.push(deploymentBytecode);
        bytecodes.push(runtimeBytecode);
    }
    return bytecodes;
}
function decodeEvmBytecode(contract, solcVersion, isDeployment, compilerBytecode, fileIdToSourceFile) {
    const libraryAddressPositions = (0, library_utils_1.getLibraryAddressPositions)(compilerBytecode);
    const immutableReferences = compilerBytecode.immutableReferences !== undefined
        ? Object.values(compilerBytecode.immutableReferences).reduce((previousValue, currentValue) => [...previousValue, ...currentValue], [])
        : [];
    const normalizedCode = (0, library_utils_1.normalizeCompilerOutputBytecode)(compilerBytecode.object, libraryAddressPositions);
    const instructions = (0, source_maps_1.decodeInstructions)(normalizedCode, compilerBytecode.sourceMap, fileIdToSourceFile, isDeployment);
    return new model_1.Bytecode(contract, isDeployment, normalizedCode, instructions, libraryAddressPositions, immutableReferences, solcVersion);
}
function astSrcToSourceLocation(src, fileIdToSourceFile) {
    const [offset, length, fileId] = src.split(":").map((p) => +p);
    const file = fileIdToSourceFile.get(fileId);
    if (file === undefined) {
        return undefined;
    }
    return new model_1.SourceLocation(file, offset, length);
}
function contractKindToContractType(contractKind) {
    if (contractKind === "library") {
        return model_1.ContractType.LIBRARY;
    }
    if (contractKind === "contract") {
        return model_1.ContractType.CONTRACT;
    }
    return undefined;
}
function astVisibilityToVisibility(visibility) {
    if (visibility === "private") {
        return model_1.ContractFunctionVisibility.PRIVATE;
    }
    if (visibility === "internal") {
        return model_1.ContractFunctionVisibility.INTERNAL;
    }
    if (visibility === "public") {
        return model_1.ContractFunctionVisibility.PUBLIC;
    }
    return model_1.ContractFunctionVisibility.EXTERNAL;
}
function functionDefinitionKindToFunctionType(kind) {
    if (kind === "constructor") {
        return model_1.ContractFunctionType.CONSTRUCTOR;
    }
    if (kind === "fallback") {
        return model_1.ContractFunctionType.FALLBACK;
    }
    if (kind === "receive") {
        return model_1.ContractFunctionType.RECEIVE;
    }
    if (kind === "freeFunction") {
        return model_1.ContractFunctionType.FREE_FUNCTION;
    }
    return model_1.ContractFunctionType.FUNCTION;
}
function astFunctionDefinitionToSelector(functionDefinition) {
    const paramTypes = [];
    // The function selector is available in solc versions >=0.6.0
    if (functionDefinition.functionSelector !== undefined) {
        return Buffer.from(functionDefinition.functionSelector, "hex");
    }
    for (const param of functionDefinition.parameters.parameters) {
        if (isContractType(param)) {
            paramTypes.push("address");
            continue;
        }
        // TODO: implement ABIv2 structs parsing
        // This might mean we need to parse struct definitions before
        // resolving types and trying to calculate function selectors.
        // if (isStructType(param)) {
        //   paramTypes.push(something);
        //   continue;
        // }
        if (isEnumType(param)) {
            // TODO: If the enum has >= 256 elements this will fail. It should be a uint16. This is
            //  complicated, as enums can be inherited. Fortunately, if multiple parent contracts
            //  define the same enum, solc fails to compile.
            paramTypes.push("uint8");
            continue;
        }
        // The rest of the function parameters always have their typeName node defined
        const typename = param.typeName;
        if (typename.nodeType === "ArrayTypeName" ||
            typename.nodeType === "FunctionTypeName" ||
            typename.nodeType === "Mapping") {
            paramTypes.push(typename.typeDescriptions.typeString);
            continue;
        }
        paramTypes.push(toCanonicalAbiType(typename.name));
    }
    return abi.methodID(functionDefinition.name, paramTypes);
}
function isContractType(param) {
    return ((param.typeName?.nodeType === "UserDefinedTypeName" ||
        param?.nodeType === "UserDefinedTypeName") &&
        param.typeDescriptions?.typeString !== undefined &&
        param.typeDescriptions.typeString.startsWith("contract "));
}
function isEnumType(param) {
    return ((param.typeName?.nodeType === "UserDefinedTypeName" ||
        param?.nodeType === "UserDefinedTypeName") &&
        param.typeDescriptions?.typeString !== undefined &&
        param.typeDescriptions.typeString.startsWith("enum "));
}
function isElementaryType(param) {
    return (param.type === "ElementaryTypeName" ||
        param.nodeType === "ElementaryTypeName");
}
function toCanonicalAbiType(type) {
    if (type.startsWith("int[")) {
        return `int256${type.slice(3)}`;
    }
    if (type === "int") {
        return "int256";
    }
    if (type.startsWith("uint[")) {
        return `uint256${type.slice(4)}`;
    }
    if (type === "uint") {
        return "uint256";
    }
    if (type.startsWith("fixed[")) {
        return `fixed128x128${type.slice(5)}`;
    }
    if (type === "fixed") {
        return "fixed128x128";
    }
    if (type.startsWith("ufixed[")) {
        return `ufixed128x128${type.slice(6)}`;
    }
    if (type === "ufixed") {
        return "ufixed128x128";
    }
    return type;
}
function correctSelectors(bytecodes, compilerOutput) {
    for (const bytecode of bytecodes) {
        if (bytecode.isDeployment) {
            continue;
        }
        const contract = bytecode.contract;
        const methodIdentifiers = compilerOutput.contracts[contract.location.file.sourceName][contract.name]
            .evm.methodIdentifiers;
        for (const [signature, hexSelector] of Object.entries(methodIdentifiers)) {
            const functionName = signature.slice(0, signature.indexOf("("));
            const selector = Buffer.from(hexSelector, "hex");
            const contractFunction = contract.getFunctionFromSelector(selector);
            if (contractFunction !== undefined) {
                continue;
            }
            const fixedSelector = contract.correctSelector(functionName, selector);
            if (!fixedSelector) {
                // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
                throw new Error(`Failed to compute the selector one or more implementations of ${contract.name}#${functionName}. Hardhat Network can automatically fix this problem if you don't use function overloading.`);
            }
        }
    }
}
//# sourceMappingURL=compiler-to-model.js.map