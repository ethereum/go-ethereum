"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ProviderWrapper = void 0;
const event_emitter_1 = require("../../util/event-emitter");
const errors_1 = require("./errors");
/**
 * A wrapper class that makes it easy to implement the EIP1193 (Javascript Ethereum Provider) standard.
 * It comes baked in with all EventEmitter methods needed,
 * which will be added to the provider supplied in the constructor.
 * It also provides the interface for the standard .request() method as an abstract method.
 */
class ProviderWrapper extends event_emitter_1.EventEmitterWrapper {
    constructor(_wrappedProvider) {
        super(_wrappedProvider);
        this._wrappedProvider = _wrappedProvider;
    }
    /**
     * Extract the params from RequestArguments and optionally type them.
     * It defaults to an empty array if no params are found.
     */
    _getParams(args) {
        const params = args.params;
        if (params === undefined) {
            return [];
        }
        if (!Array.isArray(params)) {
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw new errors_1.InvalidInputError("Hardhat Network doesn't support JSON-RPC params sent as an object");
        }
        return params;
    }
}
exports.ProviderWrapper = ProviderWrapper;
//# sourceMappingURL=wrapper.js.map