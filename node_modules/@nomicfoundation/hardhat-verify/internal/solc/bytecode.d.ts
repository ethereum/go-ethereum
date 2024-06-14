import type { CompilerOutputBytecode, EthereumProvider } from "hardhat/types";
export declare class Bytecode {
    private _bytecode;
    private _version;
    private _executableSection;
    private _isOvm;
    constructor(bytecode: string);
    static getDeployedContractBytecode(address: string, provider: EthereumProvider, network: string): Promise<Bytecode>;
    stringify(): string;
    getVersion(): string;
    isOvm(): boolean;
    hasVersionRange(): boolean;
    getMatchingVersions(versions: string[]): Promise<string[]>;
    /**
     * Compare the bytecode against a compiler's output bytecode, ignoring metadata.
     */
    compare(compilerOutputDeployedBytecode: CompilerOutputBytecode): boolean;
    private _getExecutableSection;
}
//# sourceMappingURL=bytecode.d.ts.map