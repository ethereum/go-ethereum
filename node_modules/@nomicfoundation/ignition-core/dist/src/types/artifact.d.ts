/**
 * The type of the contract artifact's ABI.
 *
 * @beta
 */
export type Abi = readonly any[] | any[];
/**
 * An compilation artifact representing a smart contract.
 *
 * @beta
 */
export interface Artifact<AbiT extends Abi = Abi> {
    contractName: string;
    sourceName: string;
    bytecode: string;
    abi: AbiT;
    linkReferences: Record<string, Record<string, Array<{
        length: number;
        start: number;
    }>>>;
}
/**
 * Retrieve artifacts based on contract name.
 *
 * @beta
 */
export interface ArtifactResolver {
    loadArtifact(contractName: string): Promise<Artifact>;
    getBuildInfo(contractName: string): Promise<BuildInfo | undefined>;
}
/**
 * A BuildInfo is a file that contains all the information of a solc run. It
 * includes all the necessary information to recreate that exact same run, and
 * all of its output.
 *
 * @beta
 */
export interface BuildInfo {
    _format: string;
    id: string;
    solcVersion: string;
    solcLongVersion: string;
    input: CompilerInput;
    output: CompilerOutput;
}
/**
 * The solc input for running the compilation.
 *
 * @beta
 */
export interface CompilerInput {
    language: string;
    sources: {
        [sourceName: string]: {
            content: string;
        };
    };
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
        metadata?: {
            useLiteralContent: boolean;
        };
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
/**
 * The output of a compiled contract from solc.
 *
 * @beta
 */
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
/**
 * The compilation output from solc.
 *
 * @beta
 */
export interface CompilerOutput {
    sources: CompilerOutputSources;
    contracts: {
        [sourceName: string]: {
            [contractName: string]: CompilerOutputContract;
        };
    };
}
/**
 * The ast for a compiled contract.
 *
 * @beta
 */
export interface CompilerOutputSource {
    id: number;
    ast: any;
}
/**
 * The asts for the compiled contracts.
 *
 * @beta
 */
export interface CompilerOutputSources {
    [sourceName: string]: CompilerOutputSource;
}
/**
 * The solc bytecode output.
 *
 * @beta
 */
export interface CompilerOutputBytecode {
    object: string;
    opcodes: string;
    sourceMap: string;
    linkReferences: {
        [sourceName: string]: {
            [libraryName: string]: Array<{
                start: number;
                length: 20;
            }>;
        };
    };
    immutableReferences?: {
        [key: string]: Array<{
            start: number;
            length: number;
        }>;
    };
}
//# sourceMappingURL=artifact.d.ts.map