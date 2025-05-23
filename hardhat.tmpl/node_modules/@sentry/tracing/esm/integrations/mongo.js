import { __read, __spread } from "tslib";
import { dynamicRequire, fill, logger } from '@sentry/utils';
var OPERATIONS = [
    'aggregate',
    'bulkWrite',
    'countDocuments',
    'createIndex',
    'createIndexes',
    'deleteMany',
    'deleteOne',
    'distinct',
    'drop',
    'dropIndex',
    'dropIndexes',
    'estimatedDocumentCount',
    'findOne',
    'findOneAndDelete',
    'findOneAndReplace',
    'findOneAndUpdate',
    'indexes',
    'indexExists',
    'indexInformation',
    'initializeOrderedBulkOp',
    'insertMany',
    'insertOne',
    'isCapped',
    'mapReduce',
    'options',
    'parallelCollectionScan',
    'rename',
    'replaceOne',
    'stats',
    'updateMany',
    'updateOne',
];
// All of the operations above take `options` and `callback` as their final parameters, but some of them
// take additional parameters as well. For those operations, this is a map of
// { <operation name>:  [<names of additional parameters>] }, as a way to know what to call the operation's
// positional arguments when we add them to the span's `data` object later
var OPERATION_SIGNATURES = {
    // aggregate intentionally not included because `pipeline` arguments are too complex to serialize well
    // see https://github.com/getsentry/sentry-javascript/pull/3102
    bulkWrite: ['operations'],
    countDocuments: ['query'],
    createIndex: ['fieldOrSpec'],
    createIndexes: ['indexSpecs'],
    deleteMany: ['filter'],
    deleteOne: ['filter'],
    distinct: ['key', 'query'],
    dropIndex: ['indexName'],
    findOne: ['query'],
    findOneAndDelete: ['filter'],
    findOneAndReplace: ['filter', 'replacement'],
    findOneAndUpdate: ['filter', 'update'],
    indexExists: ['indexes'],
    insertMany: ['docs'],
    insertOne: ['doc'],
    mapReduce: ['map', 'reduce'],
    rename: ['newName'],
    replaceOne: ['filter', 'doc'],
    updateMany: ['filter', 'update'],
    updateOne: ['filter', 'update'],
};
/** Tracing integration for mongo package */
var Mongo = /** @class */ (function () {
    /**
     * @inheritDoc
     */
    function Mongo(options) {
        if (options === void 0) { options = {}; }
        /**
         * @inheritDoc
         */
        this.name = Mongo.id;
        this._operations = Array.isArray(options.operations)
            ? options.operations
            : OPERATIONS;
        this._describeOperations = 'describeOperations' in options ? options.describeOperations : true;
    }
    /**
     * @inheritDoc
     */
    Mongo.prototype.setupOnce = function (_, getCurrentHub) {
        var collection;
        try {
            var mongodbModule = dynamicRequire(module, 'mongodb');
            collection = mongodbModule.Collection;
        }
        catch (e) {
            logger.error('Mongo Integration was unable to require `mongodb` package.');
            return;
        }
        this._instrumentOperations(collection, this._operations, getCurrentHub);
    };
    /**
     * Patches original collection methods
     */
    Mongo.prototype._instrumentOperations = function (collection, operations, getCurrentHub) {
        var _this = this;
        operations.forEach(function (operation) { return _this._patchOperation(collection, operation, getCurrentHub); });
    };
    /**
     * Patches original collection to utilize our tracing functionality
     */
    Mongo.prototype._patchOperation = function (collection, operation, getCurrentHub) {
        if (!(operation in collection.prototype))
            return;
        var getSpanContext = this._getSpanContextFromOperationArguments.bind(this);
        fill(collection.prototype, operation, function (orig) {
            return function () {
                var args = [];
                for (var _i = 0; _i < arguments.length; _i++) {
                    args[_i] = arguments[_i];
                }
                var _a, _b, _c;
                var lastArg = args[args.length - 1];
                var scope = getCurrentHub().getScope();
                var parentSpan = (_a = scope) === null || _a === void 0 ? void 0 : _a.getSpan();
                // Check if the operation was passed a callback. (mapReduce requires a different check, as
                // its (non-callback) arguments can also be functions.)
                if (typeof lastArg !== 'function' || (operation === 'mapReduce' && args.length === 2)) {
                    var span_1 = (_b = parentSpan) === null || _b === void 0 ? void 0 : _b.startChild(getSpanContext(this, operation, args));
                    return orig.call.apply(orig, __spread([this], args)).then(function (res) {
                        var _a;
                        (_a = span_1) === null || _a === void 0 ? void 0 : _a.finish();
                        return res;
                    });
                }
                var span = (_c = parentSpan) === null || _c === void 0 ? void 0 : _c.startChild(getSpanContext(this, operation, args.slice(0, -1)));
                return orig.call.apply(orig, __spread([this], args.slice(0, -1), [function (err, result) {
                        var _a;
                        (_a = span) === null || _a === void 0 ? void 0 : _a.finish();
                        lastArg(err, result);
                    }]));
            };
        });
    };
    /**
     * Form a SpanContext based on the user input to a given operation.
     */
    Mongo.prototype._getSpanContextFromOperationArguments = function (collection, operation, args) {
        var data = {
            collectionName: collection.collectionName,
            dbName: collection.dbName,
            namespace: collection.namespace,
        };
        var spanContext = {
            op: "db",
            description: operation,
            data: data,
        };
        // If the operation takes no arguments besides `options` and `callback`, or if argument
        // collection is disabled for this operation, just return early.
        var signature = OPERATION_SIGNATURES[operation];
        var shouldDescribe = Array.isArray(this._describeOperations)
            ? this._describeOperations.includes(operation)
            : this._describeOperations;
        if (!signature || !shouldDescribe) {
            return spanContext;
        }
        try {
            // Special case for `mapReduce`, as the only one accepting functions as arguments.
            if (operation === 'mapReduce') {
                var _a = __read(args, 2), map = _a[0], reduce = _a[1];
                data[signature[0]] = typeof map === 'string' ? map : map.name || '<anonymous>';
                data[signature[1]] = typeof reduce === 'string' ? reduce : reduce.name || '<anonymous>';
            }
            else {
                for (var i = 0; i < signature.length; i++) {
                    data[signature[i]] = JSON.stringify(args[i]);
                }
            }
        }
        catch (_oO) {
            // no-empty
        }
        return spanContext;
    };
    /**
     * @inheritDoc
     */
    Mongo.id = 'Mongo';
    return Mongo;
}());
export { Mongo };
//# sourceMappingURL=mongo.js.map