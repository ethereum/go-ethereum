"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ContractsIdentifier = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const library_utils_1 = require("./library-utils");
const opcodes_1 = require("./opcodes");
/**
 * This class represent a somewhat special Trie of bytecodes.
 *
 * What makes it special is that every node has a set of all of its descendants and its depth.
 */
class BytecodeTrie {
    static isBytecodeTrie(o) {
        if (o === undefined || o === null) {
            return false;
        }
        return "childNodes" in o;
    }
    constructor(depth) {
        this.depth = depth;
        this.childNodes = new Map();
        this.descendants = [];
    }
    add(bytecode) {
        // eslint-disable-next-line @typescript-eslint/no-this-alias
        let trieNode = this;
        for (let currentCodeByte = 0; currentCodeByte <= bytecode.normalizedCode.length; currentCodeByte += 1) {
            if (currentCodeByte === bytecode.normalizedCode.length) {
                // If multiple contracts with the exact same bytecode are added we keep the last of them.
                // Note that this includes the metadata hash, so the chances of happening are pretty remote,
                // except in super artificial cases that we have in our test suite.
                trieNode.match = bytecode;
                return;
            }
            const byte = bytecode.normalizedCode[currentCodeByte];
            trieNode.descendants.push(bytecode);
            let childNode = trieNode.childNodes.get(byte);
            if (childNode === undefined) {
                childNode = new BytecodeTrie(currentCodeByte);
                trieNode.childNodes.set(byte, childNode);
            }
            trieNode = childNode;
        }
    }
    /**
     * Searches for a bytecode. If it's an exact match, it is returned. If there's no match, but a
     * prefix of the code is found in the trie, the node of the longest prefix is returned. If the
     * entire code is covered by the trie, and there's no match, we return undefined.
     */
    search(code, currentCodeByte = 0) {
        if (currentCodeByte > code.length) {
            return undefined;
        }
        // eslint-disable-next-line @typescript-eslint/no-this-alias
        let trieNode = this;
        for (; currentCodeByte <= code.length; currentCodeByte += 1) {
            if (currentCodeByte === code.length) {
                return trieNode.match;
            }
            const childNode = trieNode.childNodes.get(code[currentCodeByte]);
            if (childNode === undefined) {
                return trieNode;
            }
            trieNode = childNode;
        }
    }
}
class ContractsIdentifier {
    constructor(_enableCache = true) {
        this._enableCache = _enableCache;
        this._trie = new BytecodeTrie(-1);
        this._cache = new Map();
    }
    addBytecode(bytecode) {
        this._trie.add(bytecode);
        this._cache.clear();
    }
    getBytecodeForCall(code, isCreate) {
        const normalizedCode = (0, library_utils_1.normalizeLibraryRuntimeBytecodeIfNecessary)(code);
        let normalizedCodeHex;
        if (this._enableCache) {
            normalizedCodeHex = (0, ethereumjs_util_1.bytesToHex)(normalizedCode);
            const cached = this._cache.get(normalizedCodeHex);
            if (cached !== undefined) {
                return cached;
            }
        }
        const result = this._searchBytecode(isCreate, normalizedCode);
        if (this._enableCache) {
            if (result !== undefined) {
                this._cache.set(normalizedCodeHex, result);
            }
        }
        return result;
    }
    _searchBytecode(isCreate, code, normalizeLibraries = true, trie = this._trie, firstByteToSearch = 0) {
        const searchResult = trie.search(code, firstByteToSearch);
        if (searchResult === undefined) {
            return undefined;
        }
        if (!BytecodeTrie.isBytecodeTrie(searchResult)) {
            return searchResult;
        }
        // Deployment messages have their abi-encoded arguments at the end of the bytecode.
        //
        // We don't know how long those arguments are, as we don't know which contract is being
        // deployed, hence we don't know the signature of its constructor.
        //
        // To make things even harder, we can't trust that the user actually passed the right
        // amount of arguments.
        //
        // Luckily, the chances of a complete deployment bytecode being the prefix of another one are
        // remote. For example, most of the time it ends with its metadata hash, which will differ.
        //
        // We take advantage of this last observation, and just return the bytecode that exactly
        // matched the searchResult (sub)trie that we got.
        if (isCreate &&
            searchResult.match !== undefined &&
            searchResult.match.isDeployment) {
            return searchResult.match;
        }
        if (normalizeLibraries) {
            for (const bytecodeWithLibraries of searchResult.descendants) {
                if (bytecodeWithLibraries.libraryAddressPositions.length === 0 &&
                    bytecodeWithLibraries.immutableReferences.length === 0) {
                    continue;
                }
                const normalizedLibrariesCode = (0, library_utils_1.zeroOutAddresses)(code, bytecodeWithLibraries.libraryAddressPositions);
                const normalizedCode = (0, library_utils_1.zeroOutSlices)(normalizedLibrariesCode, bytecodeWithLibraries.immutableReferences);
                const normalizedResult = this._searchBytecode(isCreate, normalizedCode, false, searchResult, searchResult.depth + 1);
                if (normalizedResult !== undefined) {
                    return normalizedResult;
                }
            }
        }
        // If we got here we may still have the contract, but with a different metadata hash.
        //
        // We check if we got to match the entire executable bytecode, and are just stuck because
        // of the metadata. If that's the case, we can assume that any descendant will be a valid
        // Bytecode, so we just choose the most recently added one.
        //
        // The reason this works is because there's no chance that Solidity includes an entire
        // bytecode (i.e. with metadata), as a prefix of another one.
        if (this._isMatchingMetadata(code, searchResult.depth) &&
            searchResult.descendants.length > 0) {
            return searchResult.descendants[searchResult.descendants.length - 1];
        }
        return undefined;
    }
    /**
     * Returns true if the lastByte is placed right when the metadata starts or after it.
     */
    _isMatchingMetadata(code, lastByte) {
        for (let byte = 0; byte < lastByte;) {
            const opcode = code[byte];
            // Solidity always emits REVERT INVALID right before the metadata
            if (opcode === opcodes_1.Opcode.REVERT && code[byte + 1] === opcodes_1.Opcode.INVALID) {
                return true;
            }
            byte += (0, opcodes_1.getOpcodeLength)(opcode);
        }
        return false;
    }
}
exports.ContractsIdentifier = ContractsIdentifier;
//# sourceMappingURL=contracts-identifier.js.map