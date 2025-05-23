import { EIP1193Provider, RequestArguments } from "../../../types";
import { ProviderWrapper } from "./wrapper";
export declare abstract class ProviderWrapperWithChainId extends ProviderWrapper {
    private _chainId;
    protected _getChainId(): Promise<number>;
    private _getChainIdFromEthChainId;
    private _getChainIdFromEthNetVersion;
}
export declare class ChainIdValidatorProvider extends ProviderWrapperWithChainId {
    private readonly _expectedChainId;
    private _alreadyValidated;
    constructor(provider: EIP1193Provider, _expectedChainId: number);
    request(args: RequestArguments): Promise<unknown>;
}
//# sourceMappingURL=chainId.d.ts.map