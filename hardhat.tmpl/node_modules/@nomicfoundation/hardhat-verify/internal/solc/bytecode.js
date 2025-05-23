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
exports.Bytecode = void 0;
const errors_1 = require("../errors");
const metadata_1 = require("./metadata");
const artifacts_1 = require("./artifacts");
// If the compiler output bytecode is OVM bytecode, we need to make a fix to account for a bug in some versions of
// the OVM compiler. The artifact’s deployedBytecode is incorrect, but because its bytecode (initcode) is correct, when we
// actually deploy contracts, the code that ends up getting stored on chain is also correct. During verification,
// Etherscan will compile the source code, pull out the artifact’s deployedBytecode, and then perform the
// below find and replace, then check that resulting output against the code retrieved on chain from eth_getCode.
// We define the strings for that find and replace here, and use them later so we can know if the bytecode matches
// before it gets to Etherscan.
// Source: https://github.com/ethereum-optimism/optimism/blob/8d67991aba584c1703692ea46273ea8a1ef45f56/packages/contracts/src/contract-dumps.ts#L195-L204
const OVM_FIND_OPCODES = "336000905af158601d01573d60011458600c01573d6000803e3d621234565260ea61109c52";
const OVM_REPLACE_OPCODES = "336000905af158600e01573d6000803e3d6000fd5b3d6001141558600a015760016000f35b";
class Bytecode {
    constructor(bytecode) {
        this._bytecode = bytecode;
        const bytecodeBuffer = Buffer.from(bytecode, "hex");
        this._version = (0, metadata_1.inferCompilerVersion)(bytecodeBuffer);
        this._executableSection = {
            start: 0,
            length: bytecode.length - (0, metadata_1.getMetadataSectionLength)(bytecodeBuffer) * 2,
        };
        // Check if this is OVM bytecode by looking for the concatenation of the two opcodes defined here:
        // https://github.com/ethereum-optimism/optimism/blob/33cb9025f5e463525d6abe67c8457f81a87c5a24/packages/contracts/contracts/optimistic-ethereum/OVM/execution/OVM_SafetyChecker.sol#L143
        //   - This check would only fail if the EVM solidity compiler didn't use any of the following opcodes: https://github.com/ethereum-optimism/optimism/blob/c42fc0df2790a5319027393cb8fa34e4f7bb520f/packages/contracts/contracts/optimistic-ethereum/iOVM/execution/iOVM_ExecutionManager.sol#L94-L175
        //     This is the list of opcodes that calls the OVM execution manager. But the current solidity
        //     compiler seems to add REVERT in all cases, meaning it currently won't happen and this check
        //     will always be correct.
        //   - It is possible, though very unlikely, that this string appears in the bytecode of an EVM
        //     contract. As a result result, this _isOvm flag should only be used after trying to infer
        //     the solc version
        //   - We need this check because OVM bytecode has no metadata, so when verifying
        //     OVM bytecode the check in `inferSolcVersion` will always return `MISSING_METADATA_VERSION_RANGE`.
        this._isOvm = bytecode.includes(OVM_REPLACE_OPCODES);
    }
    static async getDeployedContractBytecode(address, provider, network) {
        const response = await provider.send("eth_getCode", [
            address,
            "latest",
        ]);
        const deployedBytecode = response.replace(/^0x/, "");
        if (deployedBytecode === "") {
            throw new errors_1.DeployedBytecodeNotFoundError(address, network);
        }
        return new Bytecode(deployedBytecode);
    }
    stringify() {
        return this._bytecode;
    }
    getVersion() {
        return this._version;
    }
    isOvm() {
        return this._isOvm;
    }
    hasVersionRange() {
        return (this._version === metadata_1.MISSING_METADATA_VERSION_RANGE ||
            this._version === metadata_1.SOLC_NOT_FOUND_IN_METADATA_VERSION_RANGE);
    }
    async getMatchingVersions(versions) {
        const semver = await Promise.resolve().then(() => __importStar(require("semver")));
        const matchingCompilerVersions = versions.filter((version) => semver.satisfies(version, this._version));
        return matchingCompilerVersions;
    }
    /**
     * Compare the bytecode against a compiler's output bytecode, ignoring metadata.
     */
    compare(compilerOutputDeployedBytecode) {
        // Ignore metadata since Etherscan performs a partial match.
        // See: https://ethereum.org/es/developers/docs/smart-contracts/verifying/#etherscan
        const executableSection = this._getExecutableSection();
        let referenceExecutableSection = inferExecutableSection(compilerOutputDeployedBytecode.object);
        // If this is OVM bytecode, do the required find and replace (see above comments for more info)
        if (this._isOvm) {
            referenceExecutableSection = referenceExecutableSection
                .split(OVM_FIND_OPCODES)
                .join(OVM_REPLACE_OPCODES);
        }
        if (executableSection.length !== referenceExecutableSection.length &&
            // OVM bytecode has no metadata so we ignore this comparison if operating on OVM bytecode
            !this._isOvm) {
            return false;
        }
        const normalizedBytecode = nullifyBytecodeOffsets(executableSection, compilerOutputDeployedBytecode);
        // Library hash placeholders are embedded into the bytes where the library addresses are linked.
        // We need to zero them out to compare them.
        const normalizedReferenceBytecode = nullifyBytecodeOffsets(referenceExecutableSection, compilerOutputDeployedBytecode);
        if (normalizedBytecode === normalizedReferenceBytecode) {
            return true;
        }
        return false;
    }
    _getExecutableSection() {
        const { start, length } = this._executableSection;
        return this._bytecode.slice(start, length);
    }
}
exports.Bytecode = Bytecode;
function nullifyBytecodeOffsets(bytecode, { object: referenceBytecode, linkReferences, immutableReferences, }) {
    const offsets = [
        ...(0, artifacts_1.getLibraryOffsets)(linkReferences),
        ...(0, artifacts_1.getImmutableOffsets)(immutableReferences),
        ...(0, artifacts_1.getCallProtectionOffsets)(bytecode, referenceBytecode),
    ];
    for (const { start, length } of offsets) {
        bytecode = [
            bytecode.slice(0, start * 2),
            "0".repeat(length * 2),
            bytecode.slice((start + length) * 2),
        ].join("");
    }
    return bytecode;
}
/**
 * This function returns the executable section without actually
 * decoding the whole bytecode string.
 *
 * This is useful because the runtime object emitted by the compiler
 * may contain nonhexadecimal characters due to link placeholders.
 */
function inferExecutableSection(bytecode) {
    if (bytecode.startsWith("0x")) {
        bytecode = bytecode.slice(2);
    }
    // `Buffer.from` will return a buffer that contains bytes up until the last decodable byte.
    // To work around this we'll just slice the relevant part of the bytecode.
    const metadataLengthSlice = Buffer.from(bytecode.slice(-metadata_1.METADATA_LENGTH * 2), "hex");
    // If, for whatever reason, the bytecode is so small that we can't even read two bytes off it,
    // return the size of the entire bytecode.
    if (metadataLengthSlice.length !== metadata_1.METADATA_LENGTH) {
        return bytecode;
    }
    const metadataSectionLength = (0, metadata_1.getMetadataSectionLength)(metadataLengthSlice);
    return bytecode.slice(0, bytecode.length - metadataSectionLength * 2);
}
//# sourceMappingURL=bytecode.js.map