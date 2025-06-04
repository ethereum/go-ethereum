Object.defineProperty(exports, "__esModule", { value: true });
var tslib_1 = require("tslib");
var core_1 = require("@sentry/core");
var types_1 = require("@sentry/types");
var utils_1 = require("@sentry/utils");
var parsers_1 = require("./parsers");
var transports_1 = require("./transports");
/**
 * The Sentry Node SDK Backend.
 * @hidden
 */
var NodeBackend = /** @class */ (function (_super) {
    tslib_1.__extends(NodeBackend, _super);
    function NodeBackend() {
        return _super !== null && _super.apply(this, arguments) || this;
    }
    /**
     * @inheritDoc
     */
    // eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/explicit-module-boundary-types
    NodeBackend.prototype.eventFromException = function (exception, hint) {
        var _this = this;
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        var ex = exception;
        var mechanism = {
            handled: true,
            type: 'generic',
        };
        if (!utils_1.isError(exception)) {
            if (utils_1.isPlainObject(exception)) {
                // This will allow us to group events based on top-level keys
                // which is much better than creating new group when any key/value change
                var message = "Non-Error exception captured with keys: " + utils_1.extractExceptionKeysForMessage(exception);
                core_1.getCurrentHub().configureScope(function (scope) {
                    scope.setExtra('__serialized__', utils_1.normalizeToSize(exception));
                });
                ex = (hint && hint.syntheticException) || new Error(message);
                ex.message = message;
            }
            else {
                // This handles when someone does: `throw "something awesome";`
                // We use synthesized Error here so we can extract a (rough) stack trace.
                ex = (hint && hint.syntheticException) || new Error(exception);
                ex.message = exception;
            }
            mechanism.synthetic = true;
        }
        return new utils_1.SyncPromise(function (resolve, reject) {
            return parsers_1.parseError(ex, _this._options)
                .then(function (event) {
                utils_1.addExceptionTypeValue(event, undefined, undefined);
                utils_1.addExceptionMechanism(event, mechanism);
                resolve(tslib_1.__assign(tslib_1.__assign({}, event), { event_id: hint && hint.event_id }));
            })
                .then(null, reject);
        });
    };
    /**
     * @inheritDoc
     */
    NodeBackend.prototype.eventFromMessage = function (message, level, hint) {
        var _this = this;
        if (level === void 0) { level = types_1.Severity.Info; }
        var event = {
            event_id: hint && hint.event_id,
            level: level,
            message: message,
        };
        return new utils_1.SyncPromise(function (resolve) {
            if (_this._options.attachStacktrace && hint && hint.syntheticException) {
                var stack = hint.syntheticException ? parsers_1.extractStackFromError(hint.syntheticException) : [];
                parsers_1.parseStack(stack, _this._options)
                    .then(function (frames) {
                    event.stacktrace = {
                        frames: parsers_1.prepareFramesForEvent(frames),
                    };
                    resolve(event);
                })
                    .then(null, function () {
                    resolve(event);
                });
            }
            else {
                resolve(event);
            }
        });
    };
    /**
     * @inheritDoc
     */
    NodeBackend.prototype._setupTransport = function () {
        if (!this._options.dsn) {
            // We return the noop transport here in case there is no Dsn.
            return _super.prototype._setupTransport.call(this);
        }
        var dsn = new utils_1.Dsn(this._options.dsn);
        var transportOptions = tslib_1.__assign(tslib_1.__assign(tslib_1.__assign(tslib_1.__assign(tslib_1.__assign({}, this._options.transportOptions), (this._options.httpProxy && { httpProxy: this._options.httpProxy })), (this._options.httpsProxy && { httpsProxy: this._options.httpsProxy })), (this._options.caCerts && { caCerts: this._options.caCerts })), { dsn: this._options.dsn });
        if (this._options.transport) {
            return new this._options.transport(transportOptions);
        }
        if (dsn.protocol === 'http') {
            return new transports_1.HTTPTransport(transportOptions);
        }
        return new transports_1.HTTPSTransport(transportOptions);
    };
    return NodeBackend;
}(core_1.BaseBackend));
exports.NodeBackend = NodeBackend;
//# sourceMappingURL=backend.js.map