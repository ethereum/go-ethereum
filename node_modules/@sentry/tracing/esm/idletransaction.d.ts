import { Hub } from '@sentry/hub';
import { TransactionContext } from '@sentry/types';
import { Span, SpanRecorder } from './span';
import { Transaction } from './transaction';
export declare const DEFAULT_IDLE_TIMEOUT = 1000;
/**
 * @inheritDoc
 */
export declare class IdleTransactionSpanRecorder extends SpanRecorder {
    private readonly _pushActivity;
    private readonly _popActivity;
    transactionSpanId: string;
    constructor(_pushActivity: (id: string) => void, _popActivity: (id: string) => void, transactionSpanId?: string, maxlen?: number);
    /**
     * @inheritDoc
     */
    add(span: Span): void;
}
export declare type BeforeFinishCallback = (transactionSpan: IdleTransaction, endTimestamp: number) => void;
/**
 * An IdleTransaction is a transaction that automatically finishes. It does this by tracking child spans as activities.
 * You can have multiple IdleTransactions active, but if the `onScope` option is specified, the idle transaction will
 * put itself on the scope on creation.
 */
export declare class IdleTransaction extends Transaction {
    private readonly _idleHub?;
    private readonly _idleTimeout;
    private readonly _onScope;
    activities: Record<string, boolean>;
    private _heartbeatTimer;
    private _prevHeartbeatString;
    private _heartbeatCounter;
    private _finished;
    private readonly _beforeFinishCallbacks;
    private _initTimeout;
    constructor(transactionContext: TransactionContext, _idleHub?: Hub | undefined, _idleTimeout?: number, _onScope?: boolean);
    /** {@inheritDoc} */
    finish(endTimestamp?: number): string | undefined;
    /**
     * Register a callback function that gets excecuted before the transaction finishes.
     * Useful for cleanup or if you want to add any additional spans based on current context.
     *
     * This is exposed because users have no other way of running something before an idle transaction
     * finishes.
     */
    registerBeforeFinishCallback(callback: BeforeFinishCallback): void;
    /**
     * @inheritDoc
     */
    initSpanRecorder(maxlen?: number): void;
    /**
     * Start tracking a specific activity.
     * @param spanId The span id that represents the activity
     */
    private _pushActivity;
    /**
     * Remove an activity from usage
     * @param spanId The span id that represents the activity
     */
    private _popActivity;
    /**
     * Checks when entries of this.activities are not changing for 3 beats.
     * If this occurs we finish the transaction.
     */
    private _beat;
    /**
     * Pings the heartbeat
     */
    private _pingHeartbeat;
}
//# sourceMappingURL=idletransaction.d.ts.map