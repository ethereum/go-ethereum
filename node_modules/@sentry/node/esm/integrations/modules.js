import { __assign } from "tslib";
import { existsSync, readFileSync } from 'fs';
import { dirname, join } from 'path';
var moduleCache;
/** Extract information about paths */
function getPaths() {
    try {
        return require.cache ? Object.keys(require.cache) : [];
    }
    catch (e) {
        return [];
    }
}
/** Extract information about package.json modules */
function collectModules() {
    var mainPaths = (require.main && require.main.paths) || [];
    var paths = getPaths();
    var infos = {};
    var seen = {};
    paths.forEach(function (path) {
        var dir = path;
        /** Traverse directories upward in the search of package.json file */
        var updir = function () {
            var orig = dir;
            dir = dirname(orig);
            if (!dir || orig === dir || seen[orig]) {
                return undefined;
            }
            if (mainPaths.indexOf(dir) < 0) {
                return updir();
            }
            var pkgfile = join(orig, 'package.json');
            seen[orig] = true;
            if (!existsSync(pkgfile)) {
                return updir();
            }
            try {
                var info = JSON.parse(readFileSync(pkgfile, 'utf8'));
                infos[info.name] = info.version;
            }
            catch (_oO) {
                // no-empty
            }
        };
        updir();
    });
    return infos;
}
/** Add node modules / packages to the event */
var Modules = /** @class */ (function () {
    function Modules() {
        /**
         * @inheritDoc
         */
        this.name = Modules.id;
    }
    /**
     * @inheritDoc
     */
    Modules.prototype.setupOnce = function (addGlobalEventProcessor, getCurrentHub) {
        var _this = this;
        addGlobalEventProcessor(function (event) {
            if (!getCurrentHub().getIntegration(Modules)) {
                return event;
            }
            return __assign(__assign({}, event), { modules: _this._getModules() });
        });
    };
    /** Fetches the list of modules and the versions loaded by the entry file for your node.js app. */
    Modules.prototype._getModules = function () {
        if (!moduleCache) {
            moduleCache = collectModules();
        }
        return moduleCache;
    };
    /**
     * @inheritDoc
     */
    Modules.id = 'Modules';
    return Modules;
}());
export { Modules };
//# sourceMappingURL=modules.js.map