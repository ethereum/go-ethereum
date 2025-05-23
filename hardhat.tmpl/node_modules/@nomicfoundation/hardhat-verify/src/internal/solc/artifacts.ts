import type {
  Artifacts,
  BuildInfo,
  CompilerInput,
  CompilerOutputBytecode,
  CompilerOutputContract,
  Network,
} from "hardhat/types";

import { parseFullyQualifiedName } from "hardhat/utils/contract-names";
import {
  DeployedBytecodeMismatchError,
  DeployedBytecodeMultipleMatchesError,
  DuplicatedLibraryError,
  InvalidLibraryAddressError,
  LibraryAddressesMismatchError,
  LibraryMultipleMatchesError,
  LibraryNotFoundError,
  MissingLibrariesError,
} from "../errors";
import { Bytecode } from "./bytecode";

export interface ContractInformation {
  compilerInput: CompilerInput;
  solcLongVersion: string;
  sourceName: string;
  contractName: string;
  contractOutput: CompilerOutputContract;
  deployedBytecode: string;
}

interface LibraryInformation {
  libraries: SourceToLibraryToAddress;
  undetectableLibraries: string[];
}

export type ExtendedContractInformation = ContractInformation &
  LibraryInformation;

export type LibraryToAddress = Record<string, string>;

export type SourceToLibraryToAddress = Record<string, LibraryToAddress>;

export interface ByteOffset {
  start: number;
  length: number;
}

export function getLibraryOffsets(
  linkReferences: CompilerOutputBytecode["linkReferences"] = {}
): ByteOffset[] {
  const offsets: ByteOffset[] = [];
  for (const libraries of Object.values(linkReferences)) {
    for (const libraryOffset of Object.values(libraries)) {
      offsets.push(...libraryOffset);
    }
  }
  return offsets;
}

export function getImmutableOffsets(
  immutableReferences: CompilerOutputBytecode["immutableReferences"] = {}
): ByteOffset[] {
  const offsets: ByteOffset[] = [];
  for (const immutableOffset of Object.values(immutableReferences)) {
    offsets.push(...immutableOffset);
  }
  return offsets;
}

/**
 * To normalize a library object we need to take into account its call
 * protection mechanism.
 * See https://solidity.readthedocs.io/en/latest/contracts.html#call-protection-for-libraries
 */
export function getCallProtectionOffsets(
  bytecode: string,
  referenceBytecode: string
): ByteOffset[] {
  const offsets: ByteOffset[] = [];
  const addressSize = 20;
  const push20OpcodeHex = "73";
  const pushPlaceholder = push20OpcodeHex + "0".repeat(addressSize * 2);
  if (
    referenceBytecode.startsWith(pushPlaceholder) &&
    bytecode.startsWith(push20OpcodeHex)
  ) {
    offsets.push({ start: 1, length: addressSize });
  }
  return offsets;
}

/**
 * Given a contract's fully qualified name, obtains the corresponding contract
 * information from the build-info by comparing the provided bytecode with the
 * deployed bytecode. If the bytecodes match, the function returns the contract
 * information. Otherwise, it returns null.
 */
export function extractMatchingContractInformation(
  contractFQN: string,
  buildInfo: BuildInfo,
  bytecode: Bytecode
): ContractInformation | null {
  const { sourceName, contractName } = parseFullyQualifiedName(contractFQN);
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

/**
 * Searches through the artifacts for a contract that matches the given
 * deployed bytecode. If it finds a match, the function returns the contract
 * information.
 */
export async function extractInferredContractInformation(
  artifacts: Artifacts,
  network: Network,
  matchingCompilerVersions: string[],
  bytecode: Bytecode
): Promise<ContractInformation> {
  const contractMatches = await lookupMatchingBytecode(
    artifacts,
    matchingCompilerVersions,
    bytecode
  );

  if (contractMatches.length === 0) {
    throw new DeployedBytecodeMismatchError(network.name);
  }

  if (contractMatches.length > 1) {
    const fqnMatches = contractMatches.map(
      ({ sourceName, contractName }) => `${sourceName}:${contractName}`
    );
    throw new DeployedBytecodeMultipleMatchesError(fqnMatches);
  }

  return contractMatches[0];
}

/**
 * Retrieves the libraries from the contract information and combines them
 * with the libraries provided by the user. Returns a list containing all
 * the libraries required by the contract. Additionally, it returns a list of
 * undetectable libraries for debugging purposes.
 */
export async function getLibraryInformation(
  contractInformation: ContractInformation,
  userLibraries: LibraryToAddress
): Promise<LibraryInformation> {
  const allLibraries = getLibraryFQNames(
    contractInformation.contractOutput.evm.bytecode.linkReferences
  );
  const detectableLibraries = getLibraryFQNames(
    contractInformation.contractOutput.evm.deployedBytecode.linkReferences
  );
  const undetectableLibraries = allLibraries.filter(
    (lib) => !detectableLibraries.some((detLib) => detLib === lib)
  );

  // Resolve and normalize library links given by user
  const normalizedLibraries = await normalizeLibraries(
    allLibraries,
    detectableLibraries,
    undetectableLibraries,
    userLibraries,
    contractInformation.contractName
  );

  const libraryAddresses = getLibraryAddressesFromBytecode(
    contractInformation.contractOutput.evm.deployedBytecode.linkReferences,
    contractInformation.deployedBytecode
  );

  const mergedLibraryLinks = mergeLibraries(
    normalizedLibraries,
    libraryAddresses
  );

  const mergedLibraries = getLibraryFQNames(mergedLibraryLinks);
  if (mergedLibraries.length < allLibraries.length) {
    throw new MissingLibrariesError(
      `${contractInformation.sourceName}:${contractInformation.contractName}`,
      allLibraries,
      mergedLibraries,
      undetectableLibraries
    );
  }

  return { libraries: mergedLibraryLinks, undetectableLibraries };
}

async function lookupMatchingBytecode(
  artifacts: Artifacts,
  matchingCompilerVersions: string[],
  bytecode: Bytecode
): Promise<ContractInformation[]> {
  const contractMatches: ContractInformation[] = [];
  const fqNames = await artifacts.getAllFullyQualifiedNames();

  for (const fqName of fqNames) {
    const buildInfo = await artifacts.getBuildInfo(fqName);

    if (buildInfo === undefined) {
      continue;
    }

    if (
      !matchingCompilerVersions.includes(buildInfo.solcVersion) &&
      // if OVM, we will not have matching compiler versions because we can't infer a specific OVM solc version from the bytecode
      !bytecode.isOvm()
    ) {
      continue;
    }

    const contractInformation = extractMatchingContractInformation(
      fqName,
      buildInfo,
      bytecode
    );
    if (contractInformation !== null) {
      contractMatches.push(contractInformation);
    }
  }

  return contractMatches;
}

function getLibraryFQNames(
  libraries: CompilerOutputBytecode["linkReferences"] | SourceToLibraryToAddress
): string[] {
  const libraryNames: string[] = [];
  for (const [sourceName, sourceLibraries] of Object.entries(libraries)) {
    for (const libraryName of Object.keys(sourceLibraries)) {
      libraryNames.push(`${sourceName}:${libraryName}`);
    }
  }

  return libraryNames;
}

async function normalizeLibraries(
  allLibraries: string[],
  detectableLibraries: string[],
  undetectableLibraries: string[],
  userLibraries: LibraryToAddress,
  contractName: string
): Promise<SourceToLibraryToAddress> {
  const { isAddress } = await import("@ethersproject/address");

  const libraryFQNs: Set<string> = new Set();
  const normalizedLibraries: SourceToLibraryToAddress = {};
  for (const [userLibName, userLibAddress] of Object.entries(userLibraries)) {
    if (!isAddress(userLibAddress)) {
      throw new InvalidLibraryAddressError(
        contractName,
        userLibName,
        userLibAddress
      );
    }

    const foundLibraryFQN = lookupLibrary(
      allLibraries,
      detectableLibraries,
      undetectableLibraries,
      userLibName,
      contractName
    );
    const { sourceName: foundLibSource, contractName: foundLibName } =
      parseFullyQualifiedName(foundLibraryFQN);

    // The only way for this library to be already mapped is
    // for it to be given twice in the libraries user input:
    // once as a library name and another as a fully qualified library name.
    if (libraryFQNs.has(foundLibraryFQN)) {
      throw new DuplicatedLibraryError(foundLibName, foundLibraryFQN);
    }

    libraryFQNs.add(foundLibraryFQN);
    if (normalizedLibraries[foundLibSource] === undefined) {
      normalizedLibraries[foundLibSource] = {};
    }
    normalizedLibraries[foundLibSource][foundLibName] = userLibAddress;
  }

  return normalizedLibraries;
}

function lookupLibrary(
  allLibraries: string[],
  detectableLibraries: string[],
  undetectableLibraries: string[],
  userLibraryName: string,
  contractName: string
): string {
  const matchingLibraries = allLibraries.filter(
    (lib) => lib === userLibraryName || lib.split(":")[1] === userLibraryName
  );

  if (matchingLibraries.length === 0) {
    throw new LibraryNotFoundError(
      contractName,
      userLibraryName,
      allLibraries,
      detectableLibraries,
      undetectableLibraries
    );
  }

  if (matchingLibraries.length > 1) {
    throw new LibraryMultipleMatchesError(
      contractName,
      userLibraryName,
      matchingLibraries
    );
  }

  const [foundLibraryFQN] = matchingLibraries;
  return foundLibraryFQN;
}

function getLibraryAddressesFromBytecode(
  linkReferences: CompilerOutputBytecode["linkReferences"] = {},
  bytecode: string
): SourceToLibraryToAddress {
  const sourceToLibraryToAddress: SourceToLibraryToAddress = {};
  for (const [sourceName, libs] of Object.entries(linkReferences)) {
    if (sourceToLibraryToAddress[sourceName] === undefined) {
      sourceToLibraryToAddress[sourceName] = {};
    }
    for (const [libraryName, [{ start, length }]] of Object.entries(libs)) {
      sourceToLibraryToAddress[sourceName][libraryName] = `0x${bytecode.slice(
        start * 2,
        (start + length) * 2
      )}`;
    }
  }
  return sourceToLibraryToAddress;
}

function mergeLibraries(
  normalizedLibraries: SourceToLibraryToAddress,
  detectedLibraries: SourceToLibraryToAddress
): SourceToLibraryToAddress {
  const conflicts: Array<{
    library: string;
    detectedAddress: string;
    inputAddress: string;
  }> = [];
  for (const [sourceName, libraries] of Object.entries(normalizedLibraries)) {
    for (const [libraryName, libraryAddress] of Object.entries(libraries)) {
      if (
        sourceName in detectedLibraries &&
        libraryName in detectedLibraries[sourceName]
      ) {
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
    throw new LibraryAddressesMismatchError(conflicts);
  }

  // Actual merge function, used internally
  const merge = (
    targetLibraries: SourceToLibraryToAddress,
    newLibraries: SourceToLibraryToAddress
  ) => {
    for (const [sourceName, libraries] of Object.entries(newLibraries)) {
      if (targetLibraries[sourceName] === undefined) {
        targetLibraries[sourceName] = {};
      }
      for (const [libraryName, libraryAddress] of Object.entries(libraries)) {
        targetLibraries[sourceName][libraryName] = libraryAddress;
      }
    }
  };
  const mergedLibraries: SourceToLibraryToAddress = {};
  merge(mergedLibraries, normalizedLibraries);
  merge(mergedLibraries, detectedLibraries);

  return mergedLibraries;
}
