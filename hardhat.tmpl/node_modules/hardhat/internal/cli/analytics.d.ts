type AbortAnalytics = () => void;
export declare class Analytics {
    static getInstance(telemetryConsent: boolean | undefined): Promise<Analytics>;
    private readonly _clientId;
    private readonly _enabled;
    private readonly _userType;
    private readonly _analyticsUrl;
    private readonly _apiSecret;
    private readonly _measurementId;
    private _sessionId;
    private constructor();
    /**
     * Attempt to send a hit to Google Analytics using the Measurement Protocol.
     * This function returns immediately after starting the request, returning a function for aborting it.
     * The idea is that we don't want Hardhat tasks to be slowed down by a slow network request, so
     * Hardhat can abort the request if it takes too much time.
     *
     * Trying to abort a successfully completed request is a no-op, so it's always safe to call it.
     *
     * @returns The abort function
     */
    sendTaskHit(scopeName: string | undefined, taskName: string): Promise<[AbortAnalytics, Promise<void>]>;
    sendTelemetryConsentHit(userConsent: "yes" | "no"): Promise<[AbortAnalytics, Promise<void>]>;
    private _buildTaskHitPayload;
    private _sendHit;
}
export declare function requestTelemetryConsent(): Promise<boolean | undefined>;
export {};
//# sourceMappingURL=analytics.d.ts.map