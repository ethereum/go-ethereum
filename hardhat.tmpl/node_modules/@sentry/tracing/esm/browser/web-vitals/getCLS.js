/*
 * Copyright 2020 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { bindReporter } from './lib/bindReporter';
import { initMetric } from './lib/initMetric';
import { observe } from './lib/observe';
import { onHidden } from './lib/onHidden';
export var getCLS = function (onReport, reportAllChanges) {
    if (reportAllChanges === void 0) { reportAllChanges = false; }
    var metric = initMetric('CLS', 0);
    var report;
    var entryHandler = function (entry) {
        // Only count layout shifts without recent user input.
        if (!entry.hadRecentInput) {
            metric.value += entry.value;
            metric.entries.push(entry);
            report();
        }
    };
    var po = observe('layout-shift', entryHandler);
    if (po) {
        report = bindReporter(onReport, metric, po, reportAllChanges);
        onHidden(function (_a) {
            var isUnloading = _a.isUnloading;
            po.takeRecords().map(entryHandler);
            if (isUnloading) {
                metric.isFinal = true;
            }
            report();
        });
    }
};
//# sourceMappingURL=getCLS.js.map