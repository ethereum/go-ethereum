export declare const SENTRY_DSN = "https://38ba58bb85fa409e9bb7f50d2c419bc2@o385026.ingest.sentry.io/5224869";
/**
 * This class acts as a global singleton for reporting errors through Sentry.
 */
export declare class Reporter {
    static reportError(error: Error): true | undefined;
    /**
     * Enable or disable reporting. When disabled, all calls to `reportError` are
     * no-ops.
     */
    static setEnabled(enabled: boolean): void;
    /**
     * Enable or disable verbose output. This is necessary to pass the correct
     * environment variable to the transport subprocess.
     */
    static setVerbose(verbose: boolean): void;
    /**
     * The path to the hardhat config file. We use this when files are anonymized,
     * since the hardhat config is the only file in the user's project that is not
     * anonymized.
     */
    static setConfigPath(configPath: string): void;
    /**
     * Wait until all Sentry events were sent or until `timeout` milliseconds are
     * elapsed.
     *
     * This needs to be used before calling `process.exit`, otherwise some events
     * might get lost.
     */
    static close(timeout: number): Promise<boolean>;
    static shouldReport(error: Error): boolean;
    private static _instance;
    private static _getInstance;
    private static _hasTelemetryConsent;
    enabled: boolean;
    initialized: boolean;
    verbose: boolean;
    configPath?: string;
    private constructor();
    init(): void;
}
//# sourceMappingURL=reporter.d.ts.map