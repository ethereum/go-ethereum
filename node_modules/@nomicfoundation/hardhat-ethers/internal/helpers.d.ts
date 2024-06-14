import type { ethers as EthersT } from "ethers";
import type { HardhatEthersSigner } from "../signers";
import type { DeployContractOptions, FactoryOptions } from "../types";
import { Artifact, HardhatRuntimeEnvironment } from "hardhat/types";
export declare function getSigners(hre: HardhatRuntimeEnvironment): Promise<HardhatEthersSigner[]>;
export declare function getSigner(hre: HardhatRuntimeEnvironment, address: string): Promise<HardhatEthersSigner>;
export declare function getImpersonatedSigner(hre: HardhatRuntimeEnvironment, address: string): Promise<HardhatEthersSigner>;
export declare function getContractFactory<A extends any[] = any[], I = EthersT.Contract>(hre: HardhatRuntimeEnvironment, name: string, signerOrOptions?: EthersT.Signer | FactoryOptions): Promise<EthersT.ContractFactory<A, I>>;
export declare function getContractFactory<A extends any[] = any[], I = EthersT.Contract>(hre: HardhatRuntimeEnvironment, abi: any[], bytecode: EthersT.BytesLike, signer?: EthersT.Signer): Promise<EthersT.ContractFactory<A, I>>;
export declare function getContractFactoryFromArtifact<A extends any[] = any[], I = EthersT.Contract>(hre: HardhatRuntimeEnvironment, artifact: Artifact, signerOrOptions?: EthersT.Signer | FactoryOptions): Promise<EthersT.ContractFactory<A, I>>;
export declare function getContractAt(hre: HardhatRuntimeEnvironment, nameOrAbi: string | any[], address: string | EthersT.Addressable, signer?: EthersT.Signer): Promise<EthersT.Contract>;
export declare function deployContract(hre: HardhatRuntimeEnvironment, name: string, args?: any[], signerOrOptions?: EthersT.Signer | DeployContractOptions): Promise<EthersT.Contract>;
export declare function deployContract(hre: HardhatRuntimeEnvironment, name: string, signerOrOptions?: EthersT.Signer | DeployContractOptions): Promise<EthersT.Contract>;
export declare function getContractAtFromArtifact(hre: HardhatRuntimeEnvironment, artifact: Artifact, address: string | EthersT.Addressable, signer?: EthersT.Signer): Promise<EthersT.Contract>;
//# sourceMappingURL=helpers.d.ts.map