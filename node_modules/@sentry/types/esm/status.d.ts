/** The status of an event. */
export declare enum Status {
    /** The status could not be determined. */
    Unknown = "unknown",
    /** The event was skipped due to configuration or callbacks. */
    Skipped = "skipped",
    /** The event was sent to Sentry successfully. */
    Success = "success",
    /** The client is currently rate limited and will try again later. */
    RateLimit = "rate_limit",
    /** The event could not be processed. */
    Invalid = "invalid",
    /** A server-side error ocurred during submission. */
    Failed = "failed"
}
export declare namespace Status {
    /**
     * Converts a HTTP status code into a {@link Status}.
     *
     * @param code The HTTP response status code.
     * @returns The send status or {@link Status.Unknown}.
     */
    function fromHttpCode(code: number): Status;
}
//# sourceMappingURL=status.d.ts.map