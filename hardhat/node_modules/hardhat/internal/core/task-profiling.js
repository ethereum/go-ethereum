"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.flagParallelChildren = exports.createParentTaskProfile = exports.completeTaskProfile = exports.createTaskProfile = void 0;
function createTaskProfile(name) {
    return {
        name,
        start: process.hrtime.bigint(),
        children: [],
    };
}
exports.createTaskProfile = createTaskProfile;
function completeTaskProfile(taskProfile) {
    taskProfile.end = process.hrtime.bigint();
}
exports.completeTaskProfile = completeTaskProfile;
function createParentTaskProfile(taskProfile) {
    return createTaskProfile(`super::${taskProfile.name}`);
}
exports.createParentTaskProfile = createParentTaskProfile;
/**
 * Sets `parallel` to `true` to any children that was running at the same time
 * of another.
 *
 * We assume `children[]` is in chronological `start` order.
 */
function flagParallelChildren(profile, isParentParallel = false) {
    if (isParentParallel) {
        profile.parallel = true;
        for (const child of profile.children) {
            child.parallel = true;
        }
    }
    else {
        for (const [i, child] of profile.children.entries()) {
            if (i === 0) {
                continue;
            }
            const prevChild = profile.children[i - 1];
            if (child.start < prevChild.end) {
                prevChild.parallel = true;
                child.parallel = true;
            }
        }
    }
    for (const child of profile.children) {
        flagParallelChildren(child, child.parallel);
    }
}
exports.flagParallelChildren = flagParallelChildren;
//# sourceMappingURL=task-profiling.js.map