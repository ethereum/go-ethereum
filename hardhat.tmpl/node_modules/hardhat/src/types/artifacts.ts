export interface Artifacts {
  /**
   * Reads an artifact.
   *
   * @param contractNameOrFullyQualifiedName The name of the contract.
   *   It can be a contract bare contract name (e.g. "Token") if it's
   *   unique in your project, or a fully qualified contract name
   *   (e.g. "contract/token.sol:Token") otherwise.
   *
   * @throws Throws an error if a non-unique contract name is used,
   *   indicating which fully qualified names can be used instead.
   */
  readArtifact(contractNameOrFullyQualifiedName: string): Promise<Artifact>;

  /**
   * Synchronous version of readArtifact.
   */
  readArtifactSync(contractNameOrFullyQualifiedName: string): Artifact;

  /**
   * Returns true if an artifact exists.
   *
   * This function doesn't throw if the name is not unique.
   *
   * @param contractNameOrFullyQualifiedName Contract or fully qualified name.
   *
   */
  artifactExists(contractNameOrFullyQualifiedName: string): Promise<boolean>;

  /**
   * Returns an array with the fully qualified names of all the artifacts.
   */
  getAllFullyQualifiedNames(): Promise<string[]>;

  /**
   * Returns the BuildInfo associated with the solc run that compiled a
   * contract.
   *
   * Note that if your contract hasn't been compiled with solc this method
   * can return undefined.
   */
  getBuildInfo(fullyQualifiedName: string): Promise<BuildInfo | undefined>;

  /**
   * Synchronous version of getBuildInfo.
   */
  getBuildInfoSync(fullyQualifiedName: string): BuildInfo | undefined;

  /**
   * Returns an array with the absolute paths of all the existing artifacts.
   *
   * Note that there's an artifact per contract.
   */
  getArtifactPaths(): Promise<string[]>;

  /**
   * Returns an array with the absolute paths of all the existing debug files.
   *
   * Note that there's a debug file per Solidity contract.
   */
  getDebugFilePaths(): Promise<string[]>;

  /**
   * Returns an array with the absolute paths of all the existing build infos.
   *
   * Note that there's one build info per run of solc, so they can be shared
   * by different contracts.
   */
  getBuildInfoPaths(): Promise<string[]>;

  /**
   * Saves a contract's artifact and debug file.
   *
   * @param artifact The artifact object.
   * @param pathToBuildInfo The path to the build info from the solc run that
   *  compiled the contract. If the contract was built with another compiler
   *  use `undefined` and not debug file will be saved.
   */
  saveArtifactAndDebugFile(
    artifact: Artifact,
    pathToBuildInfo?: string
  ): Promise<void>;

  /**
   * Saves the build info associated to a solc run.
   *
   * @param solcVersion The semver-compatible version number.
   * @param solcLongVersion The full solc version.
   * @param input The compiler input.
   * @param output The compiler output.
   */
  saveBuildInfo(
    solcVersion: string,
    solcLongVersion: string,
    input: CompilerInput,
    output: CompilerOutput
  ): Promise<string>;

  /**
   * Returns the absolute path to the given artifact.
   *
   * @param fullyQualifiedName The FQN of the artifact.
   */
  formArtifactPathFromFullyQualifiedName(fullyQualifiedName: string): string;

  /**
   * Starting with Hardhat 2.11.0, the artifacts object caches the information
   * about paths that it fetches from the filesystem (e.g. the list of
   * artifacts, the path that an artifact name resolves to, etc.). The artifacts
   * and buildInfos themselves are not cached, only their paths.
   *
   * This method, if present, clears that cache.
   */
  clearCache?: () => void;

  /**
   * This method, if present, disables the artifact paths cache.
   *
   * We recommend NOT using this method. The only reason it exists is for
   * backwards compatibility. If your app was assuming no cache, you can use it
   * (e.g. from an HRE extender).
   *
   * @see clearCache
   */
  disableCache?: () => void;
}

/**
 * An artifact representing the compilation output of a contract.
 *
 * This file has just enough information to deploy the contract and interact
 * with an already deployed instance of it.
 *
 * For debugging information and other extra information, you should look for
 * its companion DebugFile, which should be stored right next to it.
 *
 * Note that DebugFiles are only generated for Solidity contracts.
 */
export interface Artifact {
  _format: string;
  contractName: string;
  sourceName: string;
  abi: any[];
  bytecode: string; // "0x"-prefixed hex string
  deployedBytecode: string; // "0x"-prefixed hex string
  linkReferences: LinkReferences;
  deployedLinkReferences: LinkReferences;
}

/**
 * A DebugFile contains any extra information about a Solidity contract that
 * Hardhat and its plugins need.
 *
 * The current version of DebugFiles only contains a path to a BuildInfo file.
 */
export interface DebugFile {
  _format: string;
  buildInfo: string;
}

/**
 * A BuildInfo is a file that contains all the information of a solc run. It
 * includes all the necessary information to recreate that exact same run, and
 * all of its output.
 */
export interface BuildInfo {
  _format: string;
  id: string;
  solcVersion: string;
  solcLongVersion: string;
  input: CompilerInput;
  output: CompilerOutput;
}

export interface LinkReferences {
  [libraryFileName: string]: {
    [libraryName: string]: Array<{ length: number; start: number }>;
  };
}

export interface CompilerInput {
  language: string;
  sources: { [sourceName: string]: { content: string } };
  settings: {
    viaIR?: boolean;
    optimizer: {
      runs?: number;
      enabled?: boolean;
      details?: {
        yulDetails: {
          optimizerSteps: string;
        };
      };
    };
    metadata?: { useLiteralContent: boolean };
    outputSelection: {
      [sourceName: string]: {
        [contractName: string]: string[];
      };
    };
    evmVersion?: string;
    libraries?: {
      [libraryFileName: string]: {
        [libraryName: string]: string;
      };
    };
    remappings?: string[];
  };
}

export interface CompilerOutputContract {
  abi: any;
  evm: {
    bytecode: CompilerOutputBytecode;
    deployedBytecode: CompilerOutputBytecode;
    methodIdentifiers: {
      [methodSignature: string]: string;
    };
  };
}

export interface CompilerOutput {
  sources: CompilerOutputSources;
  contracts: {
    [sourceName: string]: {
      [contractName: string]: CompilerOutputContract;
    };
  };
}

export interface CompilerOutputSource {
  id: number;
  ast: any;
}

export interface CompilerOutputSources {
  [sourceName: string]: CompilerOutputSource;
}

export interface CompilerOutputBytecode {
  object: string;
  opcodes: string;
  sourceMap: string;
  linkReferences: {
    [sourceName: string]: {
      [libraryName: string]: Array<{ start: number; length: 20 }>;
    };
  };
  immutableReferences?: {
    [key: string]: Array<{ start: number; length: number }>;
  };
}
