import { JournalMessage } from "../../execution/types/messages";
/**
 * Store a deployments execution state as a transaction log.
 *
 * @beta
 */
export interface Journal {
    record(message: JournalMessage): void;
    read(): AsyncGenerator<JournalMessage>;
}
//# sourceMappingURL=index.d.ts.map