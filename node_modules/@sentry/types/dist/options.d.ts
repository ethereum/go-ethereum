import { Breadcrumb, BreadcrumbHint } from './breadcrumb';
import { Event, EventHint } from './event';
import { Integration } from './integration';
import { LogLevel } from './loglevel';
import { SamplingContext } from './transaction';
import { Transport, TransportClass, TransportOptions } from './transport';
/** Base configuration options for every SDK. */
export interface Options {
    /**
     * Enable debug functionality in the SDK itself
     */
    debug?: boolean;
    /**
     * Specifies whether this SDK should activate and send events to Sentry.
     * Disabling the SDK reduces all overhead from instrumentation, collecting
     * breadcrumbs and capturing events. Defaults to true.
     */
    enabled?: boolean;
    /**
     * The Dsn used to connect to Sentry and identify the project. If omitted, the
     * SDK will not send any data to Sentry.
     */
    dsn?: string;
    /**
     * If this is set to false, default integrations will not be added, otherwise this will internally be set to the
     * recommended default integrations.
     * TODO: We should consider changing this to `boolean | Integration[]`
     */
    defaultIntegrations?: false | Integration[];
    /**
     * List of integrations that should be installed after SDK was initialized.
     * Accepts either a list of integrations or a function that receives
     * default integrations and returns a new, updated list.
     */
    integrations?: Integration[] | ((integrations: Integration[]) => Integration[]);
    /**
     * A pattern for error messages which should not be sent to Sentry.
     * By default, all errors will be sent.
     */
    ignoreErrors?: Array<string | RegExp>;
    /**
     * Transport object that should be used to send events to Sentry
     */
    transport?: TransportClass<Transport>;
    /**
     * Options for the default transport that the SDK uses.
     */
    transportOptions?: TransportOptions;
    /**
     * The release identifier used when uploading respective source maps. Specify
     * this value to allow Sentry to resolve the correct source maps when
     * processing events.
     */
    release?: string;
    /** The current environment of your application (e.g. "production"). */
    environment?: string;
    /** Sets the distribution for all events */
    dist?: string;
    /**
     * The maximum number of breadcrumbs sent with events. Defaults to 100.
     * Values over 100 will be ignored and 100 used instead.
     */
    maxBreadcrumbs?: number;
    /** Console logging verbosity for the SDK Client. */
    logLevel?: LogLevel;
    /** A global sample rate to apply to all events (0 - 1). */
    sampleRate?: number;
    /** Attaches stacktraces to pure capture message / log integrations */
    attachStacktrace?: boolean;
    /** Maxium number of chars a single value can have before it will be truncated. */
    maxValueLength?: number;
    /**
     * Maximum number of levels that normalization algorithm will traverse in objects and arrays.
     * Used when normalizing an event before sending, on all of the listed attributes:
     * - `breadcrumbs.data`
     * - `user`
     * - `contexts`
     * - `extra`
     * Defaults to `3`. Set to `0` to disable.
     */
    normalizeDepth?: number;
    /**
     * Controls how many milliseconds to wait before shutting down. The default is
     * SDK-specific but typically around 2 seconds. Setting this too low can cause
     * problems for sending events from command line applications. Setting it too
     * high can cause the application to block for users with network connectivity
     * problems.
     */
    shutdownTimeout?: number;
    /**
     * Options which are in beta, or otherwise not guaranteed to be stable.
     */
    _experiments?: {
        [key: string]: any;
    };
    /**
     * Sample rate to determine trace sampling.
     *
     * 0.0 = 0% chance of a given trace being sent (send no traces) 1.0 = 100% chance of a given trace being sent (send
     * all traces)
     *
     * Tracing is enabled if either this or `tracesSampler` is defined. If both are defined, `tracesSampleRate` is
     * ignored.
     */
    tracesSampleRate?: number;
    /**
     * Function to compute tracing sample rate dynamically and filter unwanted traces.
     *
     * Tracing is enabled if either this or `tracesSampleRate` is defined. If both are defined, `tracesSampleRate` is
     * ignored.
     *
     * Will automatically be passed a context object of default and optional custom data. See
     * {@link Transaction.samplingContext} and {@link Hub.startTransaction}.
     *
     * @returns A sample rate between 0 and 1 (0 drops the trace, 1 guarantees it will be sent). Returning `true` is
     * equivalent to returning 1 and returning `false` is equivalent to returning 0.
     */
    tracesSampler?(samplingContext: SamplingContext): number | boolean;
    /**
     * A callback invoked during event submission, allowing to optionally modify
     * the event before it is sent to Sentry.
     *
     * Note that you must return a valid event from this callback. If you do not
     * wish to modify the event, simply return it at the end.
     * Returning null will cause the event to be dropped.
     *
     * @param event The error or message event generated by the SDK.
     * @param hint May contain additional information about the original exception.
     * @returns A new event that will be sent | null.
     */
    beforeSend?(event: Event, hint?: EventHint): PromiseLike<Event | null> | Event | null;
    /**
     * A callback invoked when adding a breadcrumb, allowing to optionally modify
     * it before adding it to future events.
     *
     * Note that you must return a valid breadcrumb from this callback. If you do
     * not wish to modify the breadcrumb, simply return it at the end.
     * Returning null will cause the breadcrumb to be dropped.
     *
     * @param breadcrumb The breadcrumb as created by the SDK.
     * @returns The breadcrumb that will be added | null.
     */
    beforeBreadcrumb?(breadcrumb: Breadcrumb, hint?: BreadcrumbHint): Breadcrumb | null;
}
//# sourceMappingURL=options.d.ts.map