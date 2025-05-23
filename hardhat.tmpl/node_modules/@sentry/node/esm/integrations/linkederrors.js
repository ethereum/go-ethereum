import { __read, __spread } from "tslib";
import { addGlobalEventProcessor, getCurrentHub } from '@sentry/core';
import { isInstanceOf, SyncPromise } from '@sentry/utils';
import { getExceptionFromError } from '../parsers';
var DEFAULT_KEY = 'cause';
var DEFAULT_LIMIT = 5;
/** Adds SDK info to an event. */
var LinkedErrors = /** @class */ (function () {
    /**
     * @inheritDoc
     */
    function LinkedErrors(options) {
        if (options === void 0) { options = {}; }
        /**
         * @inheritDoc
         */
        this.name = LinkedErrors.id;
        this._key = options.key || DEFAULT_KEY;
        this._limit = options.limit || DEFAULT_LIMIT;
    }
    /**
     * @inheritDoc
     */
    LinkedErrors.prototype.setupOnce = function () {
        addGlobalEventProcessor(function (event, hint) {
            var self = getCurrentHub().getIntegration(LinkedErrors);
            if (self) {
                var handler = self._handler && self._handler.bind(self);
                return typeof handler === 'function' ? handler(event, hint) : event;
            }
            return event;
        });
    };
    /**
     * @inheritDoc
     */
    LinkedErrors.prototype._handler = function (event, hint) {
        var _this = this;
        if (!event.exception || !event.exception.values || !hint || !isInstanceOf(hint.originalException, Error)) {
            return SyncPromise.resolve(event);
        }
        return new SyncPromise(function (resolve) {
            _this._walkErrorTree(hint.originalException, _this._key)
                .then(function (linkedErrors) {
                if (event && event.exception && event.exception.values) {
                    event.exception.values = __spread(linkedErrors, event.exception.values);
                }
                resolve(event);
            })
                .then(null, function () {
                resolve(event);
            });
        });
    };
    /**
     * @inheritDoc
     */
    LinkedErrors.prototype._walkErrorTree = function (error, key, stack) {
        var _this = this;
        if (stack === void 0) { stack = []; }
        if (!isInstanceOf(error[key], Error) || stack.length + 1 >= this._limit) {
            return SyncPromise.resolve(stack);
        }
        return new SyncPromise(function (resolve, reject) {
            getExceptionFromError(error[key])
                .then(function (exception) {
                _this._walkErrorTree(error[key], key, __spread([exception], stack))
                    .then(resolve)
                    .then(null, function () {
                    reject();
                });
            })
                .then(null, function () {
                reject();
            });
        });
    };
    /**
     * @inheritDoc
     */
    LinkedErrors.id = 'LinkedErrors';
    return LinkedErrors;
}());
export { LinkedErrors };
//# sourceMappingURL=linkederrors.js.map