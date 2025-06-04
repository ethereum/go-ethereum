import { Hub } from '@sentry/hub';
import { Measurements, Transaction as TransactionInterface, TransactionContext } from '@sentry/types';
import { Span as SpanClass } from './span';
/** JSDoc */
export declare class Transaction extends SpanClass implements TransactionInterface {
    name: string;
    private _measurements;
    /**
     * The reference to the current hub.
     */
    private readonly _hub;
    private readonly _trimEnd?;
    /**
     * This constructor should never be called manually. Those instrumenting tracing should use
     * `Sentry.startTransaction()`, and internal methods should use `hub.startTransaction()`.
     * @internal
     * @hideconstructor
     * @hidden
     */
    constructor(transactionContext: TransactionContext, hub?: Hub);
    /**
     * JSDoc
     */
    setName(name: string): void;
    /**
     * Attaches SpanRecorder to the span itself
     * @param maxlen maximum number of spans that can be recorded
     */
    initSpanRecorder(maxlen?: number): void;
    /**
     * Set observed measurements for this transaction.
     * @hidden
     */
    setMeasurements(measurements: Measurements): void;
    /**
     * @inheritDoc
     */
    finish(endTimestamp?: number): string | undefined;
}
//# sourceMappingURL=transaction.d.ts.map