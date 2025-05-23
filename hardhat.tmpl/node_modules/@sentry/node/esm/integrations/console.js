import { __read, __spread, __values } from "tslib";
import { getCurrentHub } from '@sentry/core';
import { Severity } from '@sentry/types';
import { fill } from '@sentry/utils';
import * as util from 'util';
/** Console module integration */
var Console = /** @class */ (function () {
    function Console() {
        /**
         * @inheritDoc
         */
        this.name = Console.id;
    }
    /**
     * @inheritDoc
     */
    Console.prototype.setupOnce = function () {
        var e_1, _a;
        var consoleModule = require('console');
        try {
            for (var _b = __values(['debug', 'info', 'warn', 'error', 'log']), _c = _b.next(); !_c.done; _c = _b.next()) {
                var level = _c.value;
                fill(consoleModule, level, createConsoleWrapper(level));
            }
        }
        catch (e_1_1) { e_1 = { error: e_1_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_1) throw e_1.error; }
        }
    };
    /**
     * @inheritDoc
     */
    Console.id = 'Console';
    return Console;
}());
export { Console };
/**
 * Wrapper function that'll be used for every console level
 */
function createConsoleWrapper(level) {
    return function consoleWrapper(originalConsoleMethod) {
        var sentryLevel;
        switch (level) {
            case 'debug':
                sentryLevel = Severity.Debug;
                break;
            case 'error':
                sentryLevel = Severity.Error;
                break;
            case 'info':
                sentryLevel = Severity.Info;
                break;
            case 'warn':
                sentryLevel = Severity.Warning;
                break;
            default:
                sentryLevel = Severity.Log;
        }
        return function () {
            if (getCurrentHub().getIntegration(Console)) {
                getCurrentHub().addBreadcrumb({
                    category: 'console',
                    level: sentryLevel,
                    message: util.format.apply(undefined, arguments),
                }, {
                    input: __spread(arguments),
                    level: level,
                });
            }
            originalConsoleMethod.apply(this, arguments);
        };
    };
}
//# sourceMappingURL=console.js.map