import { BytecodeWithLinkReferences, CodegenConfig, Contract } from 'typechain';
export declare function codegenContractTypings(contract: Contract, codegenConfig: CodegenConfig): string;
export declare function codegenContractFactory(codegenConfig: CodegenConfig, contract: Contract, abi: any, bytecode?: BytecodeWithLinkReferences): string;
export declare function codegenAbstractContractFactory(contract: Contract, abi: any, moduleSuffix: string): string;
