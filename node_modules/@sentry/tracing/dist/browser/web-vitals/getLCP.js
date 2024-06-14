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
Object.defineProperty(exports, "__esModule", { value: true });
var bindReporter_1 = require("./lib/bindReporter");
var getFirstHidden_1 = require("./lib/getFirstHidden");
var initMetric_1 = require("./lib/initMetric");
var observe_1 = require("./lib/observe");
var onHidden_1 = require("./lib/onHidden");
var whenInput_1 = require("./lib/whenInput");
exports.getLCP = function (onReport, reportAllChanges) {
    if (reportAllChanges === void 0) { reportAllChanges = false; }
    var metric = initMetric_1.initMetric('LCP');
    var firstHidden = getFirstHidden_1.getFirstHidden();
    var report;
    var entryHandler = function (entry) {
        // The startTime attribute returns the value of the renderTime if it is not 0,
        // and the value of the loadTime otherwise.
        var value = entry.startTime;
        // If the page was hidden prior to paint time of the entry,
        // ignore it and mark the metric as final, otherwise add the entry.
        if (value < firstHidden.timeStamp) {
            metric.value = value;
            metric.entries.push(entry);
        }
        else {
            metric.isFinal = true;
        }
        report();
    };
    var po = observe_1.observe('largest-contentful-paint', entryHandler);
    if (po) {
        report = bindReporter_1.bindReporter(onReport, metric, po, reportAllChanges);
        var onFinal = function () {
            if (!metric.isFinal) {
                po.takeRecords().map(entryHandler);
                metric.isFinal = true;
                report();
            }
        };
        void whenInput_1.whenInput().then(onFinal);
        onHidden_1.onHidden(onFinal, true);
    }
};
//# sourceMappingURL=getLCP.js.map