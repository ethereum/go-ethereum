import { Primitive, Span as SpanInterface, SpanContext, Transaction } from '@sentry/types';
import { SpanStatus } from './spanstatus';
/**
 * Keeps track of finished spans for a given transaction
 * @internal
 * @hideconstructor
 * @hidden
 */
export declare class SpanRecorder {
    spans: Span[];
    private readonly _maxlen;
    constructor(maxlen?: number);
    /**
     * This is just so that we don't run out of memory while recording a lot
     * of spans. At some point we just stop and flush out the start of the
     * trace tree (i.e.the first n spans with the smallest
     * start_timestamp).
     */
    add(span: Span): void;
}
/**
 * Span contains all data about a span
 */
export declare class Span implements SpanInterface {
    /**
     * @inheritDoc
     */
    traceId: string;
    /**
     * @inheritDoc
     */
    spanId: string;
    /**
     * @inheritDoc
     */
    parentSpanId?: string;
    /**
     * Internal keeper of the status
     */
    status?: SpanStatus | string;
    /**
     * @inheritDoc
     */
    sampled?: boolean;
    /**
     * Timestamp in seconds when the span was created.
     */
    startTimestamp: number;
    /**
     * Timestamp in seconds when the span ended.
     */
    endTimestamp?: number;
    /**
     * @inheritDoc
     */
    op?: string;
    /**
     * @inheritDoc
     */
    description?: string;
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
     * List of spans that were finalized
     */
    spanRecorder?: SpanRecorder;
    /**
     * @inheritDoc
     */
    transaction?: Transaction;
    /**
     * You should never call the constructor manually, always use `Sentry.startTransaction()`
     * or call `startChild()` on an existing span.
     * @internal
     * @hideconstructor
     * @hidden
     */
    constructor(spanContext?: SpanContext);
    /**
     * @inheritDoc
     * @deprecated
     */
    child(spanContext?: Pick<SpanContext, Exclude<keyof SpanContext, 'spanId' | 'sampled' | 'traceId' | 'parentSpanId'>>): Span;
    /**
     * @inheritDoc
     */
    startChild(spanContext?: Pick<SpanContext, Exclude<keyof SpanContext, 'spanId' | 'sampled' | 'traceId' | 'parentSpanId'>>): Span;
    /**
     * @inheritDoc
     */
    setTag(key: string, value: Primitive): this;
    /**
     * @inheritDoc
     */
    setData(key: string, value: any): this;
    /**
     * @inheritDoc
     */
    setStatus(value: SpanStatus): this;
    /**
     * @inheritDoc
     */
    setHttpStatus(httpStatus: number): this;
    /**
     * @inheritDoc
     */
    isSuccess(): boolean;
    /**
     * @inheritDoc
     */
    finish(endTimestamp?: number): void;
    /**
     * @inheritDoc
     */
    toTraceparent(): string;
    /**
     * @inheritDoc
     */
    getTraceContext(): {
        data?: {
            [key: string]: any;
        };
        description?: string;
        op?: string;
        parent_span_id?: string;
        span_id: string;
        status?: string;
        tags?: {
            [key: string]: Primitive;
        };
        trace_id: string;
    };
    /**
     * @inheritDoc
     */
    toJSON(): {
        data?: {
            [key: string]: any;
        };
        description?: string;
        op?: string;
        parent_span_id?: string;
        span_id: string;
        start_timestamp: number;
        status?: string;
        tags?: {
            [key: string]: Primitive;
        };
        timestamp?: number;
        trace_id: string;
    };
}
//# sourceMappingURL=span.d.ts.map