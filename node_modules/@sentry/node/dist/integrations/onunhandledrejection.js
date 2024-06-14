Object.defineProperty(exports, "__esModule", { value: true });
var core_1 = require("@sentry/core");
var utils_1 = require("@sentry/utils");
var handlers_1 = require("../handlers");
/** Global Promise Rejection handler */
var OnUnhandledRejection = /** @class */ (function () {
    /**
     * @inheritDoc
     */
    function OnUnhandledRejection(_options) {
        if (_options === void 0) { _options = { mode: 'warn' }; }
        this._options = _options;
        /**
         * @inheritDoc
         */
        this.name = OnUnhandledRejection.id;
    }
    /**
     * @inheritDoc
     */
    OnUnhandledRejection.prototype.setupOnce = function () {
        global.process.on('unhandledRejection', this.sendUnhandledPromise.bind(this));
    };
    /**
     * Send an exception with reason
     * @param reason string
     * @param promise promise
     */
    // eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types, @typescript-eslint/no-explicit-any
    OnUnhandledRejection.prototype.sendUnhandledPromise = function (reason, promise) {
        var hub = core_1.getCurrentHub();
        if (!hub.getIntegration(OnUnhandledRejection)) {
            this._handleRejection(reason);
            return;
        }
        /* eslint-disable @typescript-eslint/no-unsafe-member-access */
        var context = (promise.domain && promise.domain.sentryContext) || {};
        hub.withScope(function (scope) {
            scope.setExtra('unhandledPromiseRejection', true);
            // Preserve backwards compatibility with raven-node for now
            if (context.user) {
                scope.setUser(context.user);
            }
            if (context.tags) {
                scope.setTags(context.tags);
            }
            if (context.extra) {
                scope.setExtras(context.extra);
            }
            hub.captureException(reason, { originalException: promise });
        });
        /* eslint-disable @typescript-eslint/no-unsafe-member-access */
        this._handleRejection(reason);
    };
    /**
     * Handler for `mode` option
     */
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    OnUnhandledRejection.prototype._handleRejection = function (reason) {
        // https://github.com/nodejs/node/blob/7cf6f9e964aa00772965391c23acda6d71972a9a/lib/internal/process/promises.js#L234-L240
        var rejectionWarning = 'This error originated either by ' +
            'throwing inside of an async function without a catch block, ' +
            'or by rejecting a promise which was not handled with .catch().' +
            ' The promise rejected with the reason:';
        /* eslint-disable no-console */
        if (this._options.mode === 'warn') {
            utils_1.consoleSandbox(function () {
                console.warn(rejectionWarning);
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                console.error(reason && reason.stack ? reason.stack : reason);
            });
        }
        else if (this._options.mode === 'strict') {
            utils_1.consoleSandbox(function () {
                console.warn(rejectionWarning);
            });
            handlers_1.logAndExitProcess(reason);
        }
        /* eslint-enable no-console */
    };
    /**
     * @inheritDoc
     */
    OnUnhandledRejection.id = 'OnUnhandledRejection';
    return OnUnhandledRejection;
}());
exports.OnUnhandledRejection = OnUnhandledRejection;
//# sourceMappingURL=onunhandledrejection.js.map