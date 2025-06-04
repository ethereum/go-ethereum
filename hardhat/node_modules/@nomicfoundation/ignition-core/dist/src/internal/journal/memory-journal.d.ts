import { ExecutionEventListener } from "../../types/execution-events";
import { JournalMessage } from "../execution/types/messages";
import { Journal } from "./types";
/**
 * An in-memory journal.
 *
 * @beta
 */
export declare class MemoryJournal implements Journal {
    private _executionEventListener?;
    private _messages;
    constructor(_executionEventListener?: ExecutionEventListener | undefined);
    record(message: JournalMessage): void;
    read(): AsyncGenerator<JournalMessage>;
    private _log;
}
//# sourceMappingURL=memory-journal.d.ts.map