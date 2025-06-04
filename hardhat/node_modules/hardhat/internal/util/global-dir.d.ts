export declare function getCacheDir(): Promise<string>;
export declare function readAnalyticsId(): Promise<string | undefined>;
/**
 * This is the first way that the analytics id was saved.
 */
export declare function readFirstLegacyAnalyticsId(): Promise<string | undefined>;
/**
 * This is the same way the analytics id is saved now, but using buidler as the
 * name of the project for env-paths
 */
export declare function readSecondLegacyAnalyticsId(): Promise<string | undefined>;
export declare function writeAnalyticsId(clientId: string): Promise<void>;
export declare function getCompilersDir(): Promise<string>;
/**
 * Checks if the user has given (or refused) consent for telemetry.
 *
 * Returns undefined if it can't be determined.
 */
export declare function hasConsentedTelemetry(): boolean | undefined;
export declare function writeTelemetryConsent(consent: boolean): void;
/**
 * Checks if we have already prompted the user to install the Hardhat for VSCode extension.
 */
export declare function hasPromptedForHHVSCode(): boolean;
export declare function writePromptedForHHVSCode(): void;
export declare function getVarsFilePath(): string;
//# sourceMappingURL=global-dir.d.ts.map