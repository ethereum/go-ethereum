import type { CompilerOutputBytecode, EthereumProvider } from "hardhat/types";

import { DeployedBytecodeNotFoundError } from "../errors";
import {
  getMetadataSectionLength,
  inferCompilerVersion,
  METADATA_LENGTH,
  MISSING_METADATA_VERSION_RANGE,
  SOLC_NOT_FOUND_IN_METADATA_VERSION_RANGE,
} from "./metadata";
import {
  ByteOffset,
  getCallProtectionOffsets,
  getImmutableOffsets,
  getLibraryOffsets,
} from "./artifacts";

// If the compiler output bytecode is OVM bytecode, we need to make a fix to account for a bug in some versions of
// the OVM compiler. The artifact’s deployedBytecode is incorrect, but because its bytecode (initcode) is correct, when we
// actually deploy contracts, the code that ends up getting stored on chain is also correct. During verification,
// Etherscan will compile the source code, pull out the artifact’s deployedBytecode, and then perform the
// below find and replace, then check that resulting output against the code retrieved on chain from eth_getCode.
// We define the strings for that find and replace here, and use them later so we can know if the bytecode matches
// before it gets to Etherscan.
// Source: https://github.com/ethereum-optimism/optimism/blob/8d67991aba584c1703692ea46273ea8a1ef45f56/packages/contracts/src/contract-dumps.ts#L195-L204
const OVM_FIND_OPCODES =
  "336000905af158601d01573d60011458600c01573d6000803e3d621234565260ea61109c52";
const OVM_REPLACE_OPCODES =
  "336000905af158600e01573d6000803e3d6000fd5b3d6001141558600a015760016000f35b";

export class Bytecode {
  private _bytecode: string;
  private _version: string;
  private _executableSection: ByteOffset;
  private _isOvm: boolean;

  constructor(bytecode: string) {
    this._bytecode = bytecode;

    const bytecodeBuffer = Buffer.from(bytecode, "hex");
    this._version = inferCompilerVersion(bytecodeBuffer);
    this._executableSection = {
      start: 0,
      length: bytecode.length - getMetadataSectionLength(bytecodeBuffer) * 2,
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

  public static async getDeployedContractBytecode(
    address: string,
    provider: EthereumProvider,
    network: string
  ): Promise<Bytecode> {
    const response: string = await provider.send("eth_getCode", [
      address,
      "latest",
    ]);
    const deployedBytecode = response.replace(/^0x/, "");

    if (deployedBytecode === "") {
      throw new DeployedBytecodeNotFoundError(address, network);
    }

    return new Bytecode(deployedBytecode);
  }

  public stringify() {
    return this._bytecode;
  }

  public getVersion() {
    return this._version;
  }

  public isOvm() {
    return this._isOvm;
  }

  public hasVersionRange(): boolean {
    return (
      this._version === MISSING_METADATA_VERSION_RANGE ||
      this._version === SOLC_NOT_FOUND_IN_METADATA_VERSION_RANGE
    );
  }

  public async getMatchingVersions(versions: string[]): Promise<string[]> {
    const semver = await import("semver");

    const matchingCompilerVersions = versions.filter((version) =>
      semver.satisfies(version, this._version)
    );

    return matchingCompilerVersions;
  }

  /**
   * Compare the bytecode against a compiler's output bytecode, ignoring metadata.
   */
  public compare(
    compilerOutputDeployedBytecode: CompilerOutputBytecode
  ): boolean {
    // Ignore metadata since Etherscan performs a partial match.
    // See: https://ethereum.org/es/developers/docs/smart-contracts/verifying/#etherscan
    const executableSection = this._getExecutableSection();
    let referenceExecutableSection = inferExecutableSection(
      compilerOutputDeployedBytecode.object
    );

    // If this is OVM bytecode, do the required find and replace (see above comments for more info)
    if (this._isOvm) {
      referenceExecutableSection = referenceExecutableSection
        .split(OVM_FIND_OPCODES)
        .join(OVM_REPLACE_OPCODES);
    }

    if (
      executableSection.length !== referenceExecutableSection.length &&
      // OVM bytecode has no metadata so we ignore this comparison if operating on OVM bytecode
      !this._isOvm
    ) {
      return false;
    }

    const normalizedBytecode = nullifyBytecodeOffsets(
      executableSection,
      compilerOutputDeployedBytecode
    );

    // Library hash placeholders are embedded into the bytes where the library addresses are linked.
    // We need to zero them out to compare them.
    const normalizedReferenceBytecode = nullifyBytecodeOffsets(
      referenceExecutableSection,
      compilerOutputDeployedBytecode
    );

    if (normalizedBytecode === normalizedReferenceBytecode) {
      return true;
    }

    return false;
  }

  private _getExecutableSection(): string {
    const { start, length } = this._executableSection;
    return this._bytecode.slice(start, length);
  }
}

function nullifyBytecodeOffsets(
  bytecode: string,
  {
    object: referenceBytecode,
    linkReferences,
    immutableReferences,
  }: CompilerOutputBytecode
): string {
  const offsets = [
    ...getLibraryOffsets(linkReferences),
    ...getImmutableOffsets(immutableReferences),
    ...getCallProtectionOffsets(bytecode, referenceBytecode),
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
function inferExecutableSection(bytecode: string): string {
  if (bytecode.startsWith("0x")) {
    bytecode = bytecode.slice(2);
  }

  // `Buffer.from` will return a buffer that contains bytes up until the last decodable byte.
  // To work around this we'll just slice the relevant part of the bytecode.
  const metadataLengthSlice = Buffer.from(
    bytecode.slice(-METADATA_LENGTH * 2),
    "hex"
  );

  // If, for whatever reason, the bytecode is so small that we can't even read two bytes off it,
  // return the size of the entire bytecode.
  if (metadataLengthSlice.length !== METADATA_LENGTH) {
    return bytecode;
  }

  const metadataSectionLength = getMetadataSectionLength(metadataLengthSlice);

  return bytecode.slice(0, bytecode.length - metadataSectionLength * 2);
}
