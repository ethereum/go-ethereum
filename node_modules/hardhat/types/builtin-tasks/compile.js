"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.CompilationJobCreationErrorReason = void 0;
var CompilationJobCreationErrorReason;
(function (CompilationJobCreationErrorReason) {
    CompilationJobCreationErrorReason["OTHER_ERROR"] = "other";
    CompilationJobCreationErrorReason["NO_COMPATIBLE_SOLC_VERSION_FOUND"] = "no-compatible-solc-version-found";
    CompilationJobCreationErrorReason["INCOMPATIBLE_OVERRIDEN_SOLC_VERSION"] = "incompatible-overriden-solc-version";
    CompilationJobCreationErrorReason["DIRECTLY_IMPORTS_INCOMPATIBLE_FILE"] = "directly-imports-incompatible-file";
    CompilationJobCreationErrorReason["INDIRECTLY_IMPORTS_INCOMPATIBLE_FILE"] = "indirectly-imports-incompatible-file";
})(CompilationJobCreationErrorReason = exports.CompilationJobCreationErrorReason || (exports.CompilationJobCreationErrorReason = {}));
//# sourceMappingURL=compile.js.map