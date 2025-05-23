import { Client, Options } from '@sentry/types';
/** A class object that can instantiate Client objects. */
export declare type ClientClass<F extends Client, O extends Options> = new (options: O) => F;
/**
 * Internal function to create a new SDK client instance. The client is
 * installed and then bound to the current scope.
 *
 * @param clientClass The client class to instantiate.
 * @param options Options to pass to the client.
 */
export declare function initAndBind<F extends Client, O extends Options>(clientClass: ClientClass<F, O>, options: O): void;
//# sourceMappingURL=sdk.d.ts.map