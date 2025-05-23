"use strict";
/* eslint-disable @typescript-eslint/prefer-function-type */
Object.defineProperty(exports, "__esModule", { value: true });
exports.ensure = void 0;
function ensure(condition, ErrorToThrow, ...errorArgs) {
    if (!condition) {
        throw new ErrorToThrow(...errorArgs);
    }
}
exports.ensure = ensure;
//# sourceMappingURL=utils.js.map