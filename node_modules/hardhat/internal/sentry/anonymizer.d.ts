import { Event } from "@sentry/node";
import { either } from "fp-ts";
export declare class Anonymizer {
    private _configPath?;
    constructor(_configPath?: string | undefined);
    /**
     * Given a sentry serialized exception
     * (https://develop.sentry.dev/sdk/event-payloads/exception/), return an
     * anonymized version of the event.
     */
    anonymize(event: any): either.Either<string, Event>;
    /**
     * Return the anonymized filename and a boolean indicating if the content of
     * the file should be anonymized
     */
    anonymizeFilename(filename: string): {
        anonymizedFilename: string;
        anonymizeContent: boolean;
    };
    anonymizeErrorMessage(errorMessage: string): string;
    raisedByHardhat(event: Event): boolean;
    protected _getFilePackageJsonPath(filename: string): string | null;
    private _isHardhatFile;
    private _anonymizeExceptions;
    private _anonymizeException;
    private _anonymizeStacktrace;
    private _anonymizeFrames;
    private _anonymizeFrame;
    private _anonymizeMnemonic;
}
//# sourceMappingURL=anonymizer.d.ts.map