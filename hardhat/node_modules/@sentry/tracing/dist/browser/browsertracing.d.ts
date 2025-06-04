import { Hub } from '@sentry/hub';
import { EventProcessor, Integration, Transaction, TransactionContext } from '@sentry/types';
import { RequestInstrumentationOptions } from './request';
export declare const DEFAULT_MAX_TRANSACTION_DURATION_SECONDS = 600;
/** Options for Browser Tracing integration */
export interface BrowserTracingOptions extends RequestInstrumentationOptions {
    /**
     * The time to wait in ms until the transaction will be finished. The transaction will use the end timestamp of
     * the last finished span as the endtime for the transaction.
     * Time is in ms.
     *
     * Default: 1000
     */
    idleTimeout: number;
    /**
     * Flag to enable/disable creation of `navigation` transaction on history changes.
     *
     * Default: true
     */
    startTransactionOnLocationChange: boolean;
    /**
     * Flag to enable/disable creation of `pageload` transaction on first pageload.
     *
     * Default: true
     */
    startTransactionOnPageLoad: boolean;
    /**
     * The maximum duration of a transaction before it will be marked as "deadline_exceeded".
     * If you never want to mark a transaction set it to 0.
     * Time is in seconds.
     *
     * Default: 600
     */
    maxTransactionDuration: number;
    /**
     * Flag Transactions where tabs moved to background with "cancelled". Browser background tab timing is
     * not suited towards doing precise measurements of operations. By default, we recommend that this option
     * be enabled as background transactions can mess up your statistics in nondeterministic ways.
     *
     * Default: true
     */
    markBackgroundTransactions: boolean;
    /**
     * beforeNavigate is called before a pageload/navigation transaction is created and allows users to modify transaction
     * context data, or drop the transaction entirely (by setting `sampled = false` in the context).
     *
     * Note: For legacy reasons, transactions can also be dropped by returning `undefined`.
     *
     * @param context: The context data which will be passed to `startTransaction` by default
     *
     * @returns A (potentially) modified context object, with `sampled = false` if the transaction should be dropped.
     */
    beforeNavigate?(context: TransactionContext): TransactionContext | undefined;
    /**
     * Instrumentation that creates routing change transactions. By default creates
     * pageload and navigation transactions.
     */
    routingInstrumentation<T extends Transaction>(startTransaction: (context: TransactionContext) => T | undefined, startTransactionOnPageLoad?: boolean, startTransactionOnLocationChange?: boolean): void;
}
/**
 * The Browser Tracing integration automatically instruments browser pageload/navigation
 * actions as transactions, and captures requests, metrics and errors as spans.
 *
 * The integration can be configured with a variety of options, and can be extended to use
 * any routing library. This integration uses {@see IdleTransaction} to create transactions.
 */
export declare class BrowserTracing implements Integration {
    /**
     * @inheritDoc
     */
    static id: string;
    /** Browser Tracing integration options */
    options: BrowserTracingOptions;
    /**
     * @inheritDoc
     */
    name: string;
    private _getCurrentHub?;
    private readonly _metrics;
    private readonly _emitOptionsWarning;
    constructor(_options?: Partial<BrowserTracingOptions>);
    /**
     * @inheritDoc
     */
    setupOnce(_: (callback: EventProcessor) => void, getCurrentHub: () => Hub): void;
    /** Create routing idle transaction. */
    private _createRouteTransaction;
}
/**
 * Gets transaction context from a sentry-trace meta.
 *
 * @returns Transaction context data from the header or undefined if there's no header or the header is malformed
 */
export declare function getHeaderContext(): Partial<TransactionContext> | undefined;
/** Returns the value of a meta tag */
export declare function getMetaContent(metaName: string): string | null;
//# sourceMappingURL=browsertracing.d.ts.map