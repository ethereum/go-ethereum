import { Hub } from '@sentry/hub';
import { EventProcessor, Integration } from '@sentry/types';
declare type Operation = typeof OPERATIONS[number];
declare const OPERATIONS: readonly ["aggregate", "bulkWrite", "countDocuments", "createIndex", "createIndexes", "deleteMany", "deleteOne", "distinct", "drop", "dropIndex", "dropIndexes", "estimatedDocumentCount", "findOne", "findOneAndDelete", "findOneAndReplace", "findOneAndUpdate", "indexes", "indexExists", "indexInformation", "initializeOrderedBulkOp", "insertMany", "insertOne", "isCapped", "mapReduce", "options", "parallelCollectionScan", "rename", "replaceOne", "stats", "updateMany", "updateOne"];
interface MongoOptions {
    operations?: Operation[];
    describeOperations?: boolean | Operation[];
}
/** Tracing integration for mongo package */
export declare class Mongo implements Integration {
    /**
     * @inheritDoc
     */
    static id: string;
    /**
     * @inheritDoc
     */
    name: string;
    private _operations;
    private _describeOperations?;
    /**
     * @inheritDoc
     */
    constructor(options?: MongoOptions);
    /**
     * @inheritDoc
     */
    setupOnce(_: (callback: EventProcessor) => void, getCurrentHub: () => Hub): void;
    /**
     * Patches original collection methods
     */
    private _instrumentOperations;
    /**
     * Patches original collection to utilize our tracing functionality
     */
    private _patchOperation;
    /**
     * Form a SpanContext based on the user input to a given operation.
     */
    private _getSpanContextFromOperationArguments;
}
export {};
//# sourceMappingURL=mongo.d.ts.map