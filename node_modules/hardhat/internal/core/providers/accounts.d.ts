import { EIP1193Provider, RequestArguments } from "../../../types";
import { ProviderWrapperWithChainId } from "./chainId";
import { ProviderWrapper } from "./wrapper";
export interface JsonRpcTransactionData {
    from?: string;
    to?: string;
    gas?: string | number;
    gasPrice?: string | number;
    value?: string | number;
    data?: string;
    nonce?: string | number;
}
export declare class LocalAccountsProvider extends ProviderWrapperWithChainId {
    private _addressToPrivateKey;
    constructor(provider: EIP1193Provider, localAccountsHexPrivateKeys: string[]);
    request(args: RequestArguments): Promise<unknown>;
    private _initializePrivateKeys;
    private _getPrivateKeyForAddress;
    private _getPrivateKeyForAddressOrNull;
    private _getNonce;
    private _getSignedTransaction;
}
export declare class HDWalletProvider extends LocalAccountsProvider {
    constructor(provider: EIP1193Provider, mnemonic: string, hdpath?: string, initialIndex?: number, count?: number, passphrase?: string);
}
declare abstract class SenderProvider extends ProviderWrapper {
    request(args: RequestArguments): Promise<unknown>;
    protected abstract _getSender(): Promise<string | undefined>;
}
export declare class AutomaticSenderProvider extends SenderProvider {
    private _firstAccount;
    protected _getSender(): Promise<string | undefined>;
}
export declare class FixedSenderProvider extends SenderProvider {
    private readonly _sender;
    constructor(provider: EIP1193Provider, _sender: string);
    protected _getSender(): Promise<string | undefined>;
}
export {};
//# sourceMappingURL=accounts.d.ts.map