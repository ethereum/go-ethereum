import { BytecodeWithLinkReferences, CodegenConfig, Config, Contract, FileDescription, TypeChainTarget } from 'typechain';
export default class Ethers extends TypeChainTarget {
    name: string;
    private readonly allFiles;
    private readonly outDirAbs;
    private readonly contractsWithoutBytecode;
    private readonly bytecodeCache;
    constructor(config: Config);
    transformFile(file: FileDescription): FileDescription[] | void;
    transformBinFile(file: FileDescription): FileDescription[] | void;
    transformAbiOrFullJsonFile(file: FileDescription): FileDescription[] | void;
    genContractTypingsFile(contract: Contract, codegenConfig: CodegenConfig): FileDescription;
    genContractFactoryFile(contract: Contract, abi: any, bytecode?: BytecodeWithLinkReferences): {
        path: string;
        contents: string;
    };
    afterRun(): FileDescription[];
}
