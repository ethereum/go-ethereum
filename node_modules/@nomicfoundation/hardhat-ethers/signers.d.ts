import type { BlockTag, TransactionRequest } from "ethers";
import { ethers } from "ethers";
import { HardhatEthersProvider } from "./internal/hardhat-ethers-provider";
export declare class HardhatEthersSigner implements ethers.Signer {
    private readonly _gasLimit?;
    readonly address: string;
    readonly provider: ethers.JsonRpcProvider | HardhatEthersProvider;
    static create(provider: HardhatEthersProvider, address: string): Promise<HardhatEthersSigner>;
    private constructor();
    connect(provider: ethers.JsonRpcProvider | HardhatEthersProvider): ethers.Signer;
    getNonce(blockTag?: BlockTag | undefined): Promise<number>;
    populateCall(tx: TransactionRequest): Promise<ethers.TransactionLike<string>>;
    populateTransaction(tx: TransactionRequest): Promise<ethers.TransactionLike<string>>;
    estimateGas(tx: TransactionRequest): Promise<bigint>;
    call(tx: TransactionRequest): Promise<string>;
    resolveName(name: string): Promise<string | null>;
    signTransaction(_tx: TransactionRequest): Promise<string>;
    sendTransaction(tx: TransactionRequest): Promise<ethers.TransactionResponse>;
    signMessage(message: string | Uint8Array): Promise<string>;
    signTypedData(domain: ethers.TypedDataDomain, types: Record<string, ethers.TypedDataField[]>, value: Record<string, any>): Promise<string>;
    getAddress(): Promise<string>;
    toJSON(): string;
    private _sendUncheckedTransaction;
}
export { HardhatEthersSigner as SignerWithAddress };
//# sourceMappingURL=signers.d.ts.map