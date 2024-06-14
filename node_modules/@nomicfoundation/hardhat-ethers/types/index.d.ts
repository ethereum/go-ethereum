import type * as ethers from "ethers";
import type { Artifact } from "hardhat/types";
import type { HardhatEthersProvider } from "../internal/hardhat-ethers-provider";
import type { HardhatEthersSigner } from "../signers";
export interface Libraries {
    [libraryName: string]: string | ethers.Addressable;
}
export interface FactoryOptions {
    signer?: ethers.Signer;
    libraries?: Libraries;
}
export type DeployContractOptions = FactoryOptions & ethers.Overrides;
export declare function getContractFactory<A extends any[] = any[], I = ethers.Contract>(name: string, signerOrOptions?: ethers.Signer | FactoryOptions): Promise<ethers.ContractFactory<A, I>>;
export declare function getContractFactory<A extends any[] = any[], I = ethers.Contract>(abi: any[], bytecode: ethers.BytesLike, signer?: ethers.Signer): Promise<ethers.ContractFactory<A, I>>;
export declare function deployContract(name: string, signerOrOptions?: ethers.Signer | DeployContractOptions): Promise<ethers.Contract>;
export declare function deployContract(name: string, args: any[], signerOrOptions?: ethers.Signer | DeployContractOptions): Promise<ethers.Contract>;
export declare function getContractFactoryFromArtifact<A extends any[] = any[], I = ethers.Contract>(artifact: Artifact, signerOrOptions?: ethers.Signer | FactoryOptions): Promise<ethers.ContractFactory<A, I>>;
export interface HardhatEthersHelpers {
    provider: HardhatEthersProvider;
    getContractFactory: typeof getContractFactory;
    getContractFactoryFromArtifact: typeof getContractFactoryFromArtifact;
    getContractAt: (nameOrAbi: string | any[], address: string | ethers.Addressable, signer?: ethers.Signer) => Promise<ethers.Contract>;
    getContractAtFromArtifact: (artifact: Artifact, address: string, signer?: ethers.Signer) => Promise<ethers.Contract>;
    getSigner: (address: string) => Promise<HardhatEthersSigner>;
    getSigners: () => Promise<HardhatEthersSigner[]>;
    getImpersonatedSigner: (address: string) => Promise<HardhatEthersSigner>;
    deployContract: typeof deployContract;
}
//# sourceMappingURL=index.d.ts.map