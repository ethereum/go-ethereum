import { ExtractedNodeRequestData, Primitive, WorkerLocation } from './misc';
import { Span, SpanContext } from './span';
/**
 * Interface holding Transaction-specific properties
 */
export interface TransactionContext extends SpanContext {
    /**
     * Human-readable identifier for the transaction
     */
    name: string;
    /**
     * If true, sets the end timestamp of the transaction to the highest timestamp of child spans, trimming
     * the duration of the transaction. This is useful to discard extra time in the transaction that is not
     * accounted for in child spans, like what happens in the idle transaction Tracing integration, where we finish the
     * transaction after a given "idle time" and we don't want this "idle time" to be part of the transaction.
     */
    trimEnd?: boolean;
    /**
     * If this transaction has a parent, the parent's sampling decision
     */
    parentSampled?: boolean;
}
/**
 * Data pulled from a `sentry-trace` header
 */
export declare type TraceparentData = Pick<TransactionContext, 'traceId' | 'parentSpanId' | 'parentSampled'>;
/**
 * Transaction "Class", inherits Span only has `setName`
 */
export interface Transaction extends TransactionContext, Span {
    /**
     * @inheritDoc
     */
    spanId: string;
    /**
     * @inheritDoc
     */
    traceId: string;
    /**
     * @inheritDoc
     */
    startTimestamp: number;
    /**
     * @inheritDoc
     */
    tags: {
        [key: string]: Primitive;
    };
    /**
     * @inheritDoc
     */
    data: {
        [key: string]: any;
    };
    /**
     * Set the name of the transaction
     */
    setName(name: string): void;
}
/**
 * Context data passed by the user when starting a transaction, to be used by the tracesSampler method.
 */
export interface CustomSamplingContext {
    [key: string]: any;
}
/**
 * Data passed to the `tracesSampler` function, which forms the basis for whatever decisions it might make.
 *
 * Adds default data to data provided by the user. See {@link Hub.startTransaction}
 */
export interface SamplingContext extends CustomSamplingContext {
    /**
     * Context data with which transaction being sampled was created
     */
    transactionContext: TransactionContext;
    /**
     * Sampling decision from the parent transaction, if any.
     */
    parentSampled?: boolean;
    /**
     * Object representing the URL of the current page or worker script. Passed by default in a browser or service worker
     * context.
     */
    location?: WorkerLocation;
    /**
     * Object representing the incoming request to a node server. Passed by default when using the TracingHandler.
     */
    request?: ExtractedNodeRequestData;
}
export declare type Measurements = Record<string, {
    value: number;
}>;
export declare enum TransactionSamplingMethod {
    Explicit = "explicitly_set",
    Sampler = "client_sampler",
    Rate = "client_rate",
    Inheritance = "inheritance"
}
//# sourceMappingURL=transaction.d.ts.map