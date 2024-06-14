import { EIP1193Provider, RequestArguments } from "../../../types";
import { EventEmitterWrapper } from "../../util/event-emitter";
/**
 * A wrapper class that makes it easy to implement the EIP1193 (Javascript Ethereum Provider) standard.
 * It comes baked in with all EventEmitter methods needed,
 * which will be added to the provider supplied in the constructor.
 * It also provides the interface for the standard .request() method as an abstract method.
 */
export declare abstract class ProviderWrapper extends EventEmitterWrapper implements EIP1193Provider {
    protected readonly _wrappedProvider: EIP1193Provider;
    constructor(_wrappedProvider: EIP1193Provider);
    abstract request(args: RequestArguments): Promise<unknown>;
    /**
     * Extract the params from RequestArguments and optionally type them.
     * It defaults to an empty array if no params are found.
     */
    protected _getParams<ParamsT extends any[] = any[]>(args: RequestArguments): ParamsT | [];
}
//# sourceMappingURL=wrapper.d.ts.map