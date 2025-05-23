"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.formatFatalError = void 0;
function formatFatalError(message) {
    return JSON.stringify({
        errors: [
            {
                type: 'JSONError',
                component: 'solcjs',
                severity: 'error',
                message: message,
                formattedMessage: 'Error: ' + message
            }
        ]
    });
}
exports.formatFatalError = formatFatalError;
