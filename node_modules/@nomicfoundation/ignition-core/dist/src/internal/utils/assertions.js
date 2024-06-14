"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.assertIgnitionInvariant = void 0;
const errors_1 = require("../../errors");
const errors_list_1 = require("../errors-list");
function assertIgnitionInvariant(invariant, description) {
    if (!invariant) {
        throw new errors_1.IgnitionError(errors_list_1.ERRORS.GENERAL.ASSERTION_ERROR, { description });
    }
}
exports.assertIgnitionInvariant = assertIgnitionInvariant;
//# sourceMappingURL=assertions.js.map