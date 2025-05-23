"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
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
exports.getLibraryInformation = exports.extractInferredContractInformation = exports.extractMatchingContractInformation = exports.getCallProtectionOffsets = exports.getImmutableOffsets = exports.getLibraryOffsets = void 0;
const contract_names_1 = require("hardhat/utils/contract-names");
const errors_1 = require("../errors");
function getLibraryOffsets(linkReferences = {}) {
    const offsets = [];
    for (const libraries of Object.values(linkReferences)) {
        for (const libraryOffset of Object.values(libraries)) {
            offsets.push(...libraryOffset);
        }
    }
    return offsets;
}
exports.getLibraryOffsets = getLibraryOffsets;
function getImmutableOffsets(immutableReferences = {}) {
    const offsets = [];
    for (const immutableOffset of Object.values(immutableReferences)) {
        offsets.push(...immutableOffset);
    }
    return offsets;
}
exports.getImmutableOffsets = getImmutableOffsets;
/**
 * To normalize a library object we need to take into account its call
 * protection mechanism.
 * See https://solidity.readthedocs.io/en/latest/contracts.html#call-protection-for-libraries
 */
function getCallProtectionOffsets(bytecode, referenceBytecode) {
    const offsets = [];
    const addressSize = 20;
    const push20OpcodeHex = "73";
    const pushPlaceholder = push20OpcodeHex + "0".repeat(addressSize * 2);
    if (referenceBytecode.startsWith(pushPlaceholder) &&
        bytecode.startsWith(push20OpcodeHex)) {
        offsets.push({ start: 1, length: addressSize });
    }
    return offsets;
}
exports.getCallProtectionOffsets = getCallProtectionOffsets;
/**
 * Given a contract's fully qualified name, obtains the corresponding contract
 * information from the build-info by comparing the provided bytecode with the
 * deployed bytecode. If the bytecodes match, the function returns the contract
 * information. Otherwise, it returns null.
 */
function extractMatchingContractInformation(contractFQN, buildInfo, bytecode) {
    const { sourceName, contractName } = (0, contract_names_1.parseFullyQualifiedName)(contractFQN);
    const contractOutput = buildInfo.output.contracts[sourceName][contractName];
    // Normalize deployed bytecode according to this object
    const compilerOutputDeployedBytecode = contractOutput.evm.deployedBytecode;
    if (bytecode.compare(compilerOutputDeployedBytecode)) {
        return {
            compilerInput: buildInfo.input,
            solcLongVersion: buildInfo.solcLongVersion,
            sourceName,
            contractName,
            contractOutput,
            deployedBytecode: bytecode.stringify(),
        };
    }
    return null;
}
exports.extractMatchingContractInformation = extractMatchingContractInformation;
/**
 * Searches through the artifacts for a contract that matches the given
 * deployed bytecode. If it finds a match, the function returns the contract
 * information.
 */
async function extractInferredContractInformation(artifacts, network, matchingCompilerVersions, bytecode) {
    const contractMatches = await lookupMatchingBytecode(artifacts, matchingCompilerVersions, bytecode);
    if (contractMatches.length === 0) {
        throw new errors_1.DeployedBytecodeMismatchError(network.name);
    }
    if (contractMatches.length > 1) {
        const fqnMatches = contractMatches.map(({ sourceName, contractName }) => `${sourceName}:${contractName}`);
        throw new errors_1.DeployedBytecodeMultipleMatchesError(fqnMatches);
    }
    return contractMatches[0];
}
exports.extractInferredContractInformation = extractInferredContractInformation;
/**
 * Retrieves the libraries from the contract information and combines them
 * with the libraries provided by the user. Returns a list containing all
 * the libraries required by the contract. Additionally, it returns a list of
 * undetectable libraries for debugging purposes.
 */
async function getLibraryInformation(contractInformation, userLibraries) {
    const allLibraries = getLibraryFQNames(contractInformation.contractOutput.evm.bytecode.linkReferences);
    const detectableLibraries = getLibraryFQNames(contractInformation.contractOutput.evm.deployedBytecode.linkReferences);
    const undetectableLibraries = allLibraries.filter((lib) => !detectableLibraries.some((detLib) => detLib === lib));
    // Resolve and normalize library links given by user
    const normalizedLibraries = await normalizeLibraries(allLibraries, detectableLibraries, undetectableLibraries, userLibraries, contractInformation.contractName);
    const libraryAddresses = getLibraryAddressesFromBytecode(contractInformation.contractOutput.evm.deployedBytecode.linkReferences, contractInformation.deployedBytecode);
    const mergedLibraryLinks = mergeLibraries(normalizedLibraries, libraryAddresses);
    const mergedLibraries = getLibraryFQNames(mergedLibraryLinks);
    if (mergedLibraries.length < allLibraries.length) {
        throw new errors_1.MissingLibrariesError(`${contractInformation.sourceName}:${contractInformation.contractName}`, allLibraries, mergedLibraries, undetectableLibraries);
    }
    return { libraries: mergedLibraryLinks, undetectableLibraries };
}
exports.getLibraryInformation = getLibraryInformation;
async function lookupMatchingBytecode(artifacts, matchingCompilerVersions, bytecode) {
    const contractMatches = [];
    const fqNames = await artifacts.getAllFullyQualifiedNames();
    for (const fqName of fqNames) {
        const buildInfo = await artifacts.getBuildInfo(fqName);
        if (buildInfo === undefined) {
            continue;
        }
        if (!matchingCompilerVersions.includes(buildInfo.solcVersion) &&
            // if OVM, we will not have matching compiler versions because we can't infer a specific OVM solc version from the bytecode
            !bytecode.isOvm()) {
            continue;
        }
        const contractInformation = extractMatchingContractInformation(fqName, buildInfo, bytecode);
        if (contractInformation !== null) {
            contractMatches.push(contractInformation);
        }
    }
    return contractMatches;
}
function getLibraryFQNames(libraries) {
    const libraryNames = [];
    for (const [sourceName, sourceLibraries] of Object.entries(libraries)) {
        for (const libraryName of Object.keys(sourceLibraries)) {
            libraryNames.push(`${sourceName}:${libraryName}`);
        }
    }
    return libraryNames;
}
async function normalizeLibraries(allLibraries, detectableLibraries, undetectableLibraries, userLibraries, contractName) {
    const { isAddress } = await Promise.resolve().then(() => __importStar(require("@ethersproject/address")));
    const libraryFQNs = new Set();
    const normalizedLibraries = {};
    for (const [userLibName, userLibAddress] of Object.entries(userLibraries)) {
        if (!isAddress(userLibAddress)) {
            throw new errors_1.InvalidLibraryAddressError(contractName, userLibName, userLibAddress);
        }
        const foundLibraryFQN = lookupLibrary(allLibraries, detectableLibraries, undetectableLibraries, userLibName, contractName);
        const { sourceName: foundLibSource, contractName: foundLibName } = (0, contract_names_1.parseFullyQualifiedName)(foundLibraryFQN);
        // The only way for this library to be already mapped is
        // for it to be given twice in the libraries user input:
        // once as a library name and another as a fully qualified library name.
        if (libraryFQNs.has(foundLibraryFQN)) {
            throw new errors_1.DuplicatedLibraryError(foundLibName, foundLibraryFQN);
        }
        libraryFQNs.add(foundLibraryFQN);
        if (normalizedLibraries[foundLibSource] === undefined) {
            normalizedLibraries[foundLibSource] = {};
        }
        normalizedLibraries[foundLibSource][foundLibName] = userLibAddress;
    }
    return normalizedLibraries;
}
function lookupLibrary(allLibraries, detectableLibraries, undetectableLibraries, userLibraryName, contractName) {
    const matchingLibraries = allLibraries.filter((lib) => lib === userLibraryName || lib.split(":")[1] === userLibraryName);
    if (matchingLibraries.length === 0) {
        throw new errors_1.LibraryNotFoundError(contractName, userLibraryName, allLibraries, detectableLibraries, undetectableLibraries);
    }
    if (matchingLibraries.length > 1) {
        throw new errors_1.LibraryMultipleMatchesError(contractName, userLibraryName, matchingLibraries);
    }
    const [foundLibraryFQN] = matchingLibraries;
    return foundLibraryFQN;
}
function getLibraryAddressesFromBytecode(linkReferences = {}, bytecode) {
    const sourceToLibraryToAddress = {};
    for (const [sourceName, libs] of Object.entries(linkReferences)) {
        if (sourceToLibraryToAddress[sourceName] === undefined) {
            sourceToLibraryToAddress[sourceName] = {};
        }
        for (const [libraryName, [{ start, length }]] of Object.entries(libs)) {
            sourceToLibraryToAddress[sourceName][libraryName] = `0x${bytecode.slice(start * 2, (start + length) * 2)}`;
        }
    }
    return sourceToLibraryToAddress;
}
function mergeLibraries(normalizedLibraries, detectedLibraries) {
    const conflicts = [];
    for (const [sourceName, libraries] of Object.entries(normalizedLibraries)) {
        for (const [libraryName, libraryAddress] of Object.entries(libraries)) {
            if (sourceName in detectedLibraries &&
                libraryName in detectedLibraries[sourceName]) {
                const detectedAddress = detectedLibraries[sourceName][libraryName];
                // Our detection logic encodes bytes into lowercase hex.
                if (libraryAddress.toLowerCase() !== detectedAddress.toLowerCase()) {
                    conflicts.push({
                        library: `${sourceName}:${libraryName}`,
                        detectedAddress,
                        inputAddress: libraryAddress,
                    });
                }
            }
        }
    }
    if (conflicts.length > 0) {
        throw new errors_1.LibraryAddressesMismatchError(conflicts);
    }
    // Actual merge function, used internally
    const merge = (targetLibraries, newLibraries) => {
        for (const [sourceName, libraries] of Object.entries(newLibraries)) {
            if (targetLibraries[sourceName] === undefined) {
                targetLibraries[sourceName] = {};
            }
            for (const [libraryName, libraryAddress] of Object.entries(libraries)) {
                targetLibraries[sourceName][libraryName] = libraryAddress;
            }
        }
    };
    const mergedLibraries = {};
    merge(mergedLibraries, normalizedLibraries);
    merge(mergedLibraries, detectedLibraries);
    return mergedLibraries;
}
//# sourceMappingURL=artifacts.js.map