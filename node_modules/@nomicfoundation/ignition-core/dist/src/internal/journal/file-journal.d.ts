import { ExecutionEventListener } from "../../types/execution-events";
import { JournalMessage } from "../execution/types/messages";
import { Journal } from "./types";
/**
 * A file-based journal.
 *
 * @beta
 */
export declare class FileJournal implements Journal {
    private _filePath;
    private _executionEventListener?;
    constructor(_filePath: string, _executionEventListener?: ExecutionEventListener | undefined);
    record(message: JournalMessage): void;
    read(): AsyncGenerator<JournalMessage>;
    private _appendJsonLine;
    private _log;
}
//# sourceMappingURL=file-journal.d.ts.map