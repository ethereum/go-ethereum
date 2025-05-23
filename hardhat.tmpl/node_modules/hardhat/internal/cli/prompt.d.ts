import { Dependencies, PackageManager } from "./types";
export declare function confirmRecommendedDepsInstallation(depsToInstall: Dependencies, packageManager: PackageManager): Promise<boolean>;
export declare function confirmProjectCreation(): Promise<{
    projectRoot: string;
    shouldAddGitIgnore: boolean;
}>;
export declare function confirmTelemetryConsent(): Promise<boolean | undefined>;
/**
 * true = install ext
 * false = don't install and don't ask again
 * undefined = we couldn't confirm if the extension is installed or not
 */
export declare function confirmHHVSCodeInstallation(): Promise<boolean | undefined>;
//# sourceMappingURL=prompt.d.ts.map