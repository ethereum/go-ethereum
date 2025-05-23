"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Web3BaseProvider = void 0;
const symbol = Symbol.for('web3/base-provider');
// Provider interface compatible with EIP-1193
// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1193.md
class Web3BaseProvider {
    static isWeb3Provider(provider) {
        return (provider instanceof Web3BaseProvider ||
            Boolean(provider && provider[symbol]));
    }
    // To match an object "instanceof" does not work if
    // matcher class and object is using different package versions
    // to overcome this bottleneck used this approach.
    // The symbol value for one string will always remain same regardless of package versions
    // eslint-disable-next-line class-methods-use-this
    get [symbol]() {
        return true;
    }
    /**
     * @deprecated Please use `.request` instead.
     * @param payload - Request Payload
     * @param callback - Callback
     */
    send(payload, 
    // eslint-disable-next-line @typescript-eslint/ban-types
    callback) {
        this.request(payload)
            .then(response => {
            // eslint-disable-next-line no-null/no-null
            callback(null, response);
        })
            .catch((err) => {
            callback(err);
        });
    }
    /**
     * @deprecated Please use `.request` instead.
     * @param payload - Request Payload
     */
    sendAsync(payload) {
        return __awaiter(this, void 0, void 0, function* () {
            return this.request(payload);
        });
    }
    /**
     * Modify the return type of the request method to be fully compatible with EIP-1193
     *
     * [deprecated] In the future major releases (\>= v5) all providers are supposed to be fully compatible with EIP-1193.
     * So this method will not be needed and would not be available in the future.
     *
     * @returns A new instance of the provider with the request method fully compatible with EIP-1193
     *
     * @example
     * ```ts
     * const provider = new Web3HttpProvider('http://localhost:8545');
     * const fullyCompatibleProvider = provider.asEIP1193Provider();
     * const result = await fullyCompatibleProvider.request({ method: 'eth_getBalance' });
     * console.log(result); // '0x0234c8a3397aab58' or something like that
     * ```
     */
    asEIP1193Provider() {
        // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
        const newObj = Object.create(this);
        // eslint-disable-next-line @typescript-eslint/unbound-method
        const originalRequest = newObj.request;
        newObj.request = function request(args) {
            return __awaiter(this, void 0, void 0, function* () {
                // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
                const response = (yield originalRequest(args));
                return response.result;
            });
        };
        // @ts-expect-error the property should not be available in the new object because of using Object.create(this).
        //	But it is available if we do not delete it.
        newObj.asEIP1193Provider = undefined; // to prevent the user for calling this method again
        return newObj;
    }
}
exports.Web3BaseProvider = Web3BaseProvider;
//# sourceMappingURL=web3_base_provider.js.map