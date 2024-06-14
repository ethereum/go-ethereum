import { Hub } from '@sentry/hub';
import { TransactionContext } from '@sentry/types';
import { IdleTransaction } from './idletransaction';
/**
 * Create new idle transaction.
 */
export declare function startIdleTransaction(hub: Hub, transactionContext: TransactionContext, idleTimeout?: number, onScope?: boolean): IdleTransaction;
/**
 * @private
 */
export declare function _addTracingExtensions(): void;
/**
 * This patches the global object and injects the Tracing extensions methods
 */
export declare function addExtensionMethods(): void;
//# sourceMappingURL=hubextensions.d.ts.map