"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.UiStateDeploymentStatus = exports.UiFutureStatusType = void 0;
var UiFutureStatusType;
(function (UiFutureStatusType) {
    UiFutureStatusType["UNSTARTED"] = "UNSTARTED";
    UiFutureStatusType["SUCCESS"] = "SUCCESS";
    UiFutureStatusType["TIMEDOUT"] = "TIMEDOUT";
    UiFutureStatusType["ERRORED"] = "ERRORED";
    UiFutureStatusType["HELD"] = "HELD";
})(UiFutureStatusType || (exports.UiFutureStatusType = UiFutureStatusType = {}));
var UiStateDeploymentStatus;
(function (UiStateDeploymentStatus) {
    UiStateDeploymentStatus["UNSTARTED"] = "UNSTARTED";
    UiStateDeploymentStatus["DEPLOYING"] = "DEPLOYING";
    UiStateDeploymentStatus["COMPLETE"] = "COMPLETE";
})(UiStateDeploymentStatus || (exports.UiStateDeploymentStatus = UiStateDeploymentStatus = {}));
//# sourceMappingURL=types.js.map