"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var PathReporter_1 = require("./PathReporter");
/**
 * @since 1.0.0
 * @deprecated
 */
exports.ThrowReporter = {
    report: function (validation) {
        if (validation.isLeft()) {
            throw PathReporter_1.PathReporter.report(validation).join('\n');
        }
    }
};
