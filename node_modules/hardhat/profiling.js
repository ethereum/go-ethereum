"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.adhocProfileSync = exports.adhocProfile = void 0;
const errors_1 = require("./internal/core/errors");
/**
 * Utility to create ad-hoc profiles when computing flamegraphs. You can think
 * of these as virtual tasks that execute the function `f`.
 *
 * This is an **unstable** feature, only meant for development. DO NOT use in
 * production code nor plugins.
 *
 * @param name The name of the profile. Think of it as a virtual task name.
 * @param f The function you want to profile.
 */
async function adhocProfile(name, f) {
    const globalAsAny = global;
    (0, errors_1.assertHardhatInvariant)("adhocProfile" in globalAsAny, "adhocProfile is missing. Are you running with --flamegraph?");
    return globalAsAny.adhocProfile(name, f);
}
exports.adhocProfile = adhocProfile;
/**
 * Sync version of adhocProfile
 *
 * @see adhocProfile
 */
function adhocProfileSync(name, f) {
    const globalAsAny = global;
    (0, errors_1.assertHardhatInvariant)("adhocProfileSync" in globalAsAny, "adhocProfileSync is missing. Are you running with --flamegraph?");
    return globalAsAny.adhocProfileSync(name, f);
}
exports.adhocProfileSync = adhocProfileSync;
//# sourceMappingURL=profiling.js.map