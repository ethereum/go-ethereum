import { EIP1193Provider, EthereumProvider, JsonRpcRequest, JsonRpcResponse, RequestArguments } from "../../../types";
import { EventEmitterWrapper } from "../../util/event-emitter";
/**
 * Hardhat predates the EIP1193 (Javascript Ethereum Provider) standard. It was
 * built following a draft of that spec, but then it changed completely. We
 * still need to support the draft api, but internally we use EIP1193. So we
 * use BackwardsCompatibilityProviderAdapter to wrap EIP1193 providers before
 * exposing them to the user.
 */
export declare class BackwardsCompatibilityProviderAdapter extends EventEmitterWrapper implements EthereumProvider {
    private readonly _provider;
    constructor(_provider: EIP1193Provider);
    request(args: RequestArguments): Promise<unknown>;
    send(method: string, params?: any[]): Promise<any>;
    sendAsync(payload: JsonRpcRequest, callback: (error: any, response: JsonRpcResponse) => void): void;
    private _sendJsonRpcRequest;
}
//# sourceMappingURL=backwards-compatibility.d.ts.map