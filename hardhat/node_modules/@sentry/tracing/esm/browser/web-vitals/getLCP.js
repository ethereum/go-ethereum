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
import { getFirstHidden } from './lib/getFirstHidden';
import { initMetric } from './lib/initMetric';
import { observe } from './lib/observe';
import { onHidden } from './lib/onHidden';
import { whenInput } from './lib/whenInput';
export var getLCP = function (onReport, reportAllChanges) {
    if (reportAllChanges === void 0) { reportAllChanges = false; }
    var metric = initMetric('LCP');
    var firstHidden = getFirstHidden();
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
    var po = observe('largest-contentful-paint', entryHandler);
    if (po) {
        report = bindReporter(onReport, metric, po, reportAllChanges);
        var onFinal = function () {
            if (!metric.isFinal) {
                po.takeRecords().map(entryHandler);
                metric.isFinal = true;
                report();
            }
        };
        void whenInput().then(onFinal);
        onHidden(onFinal, true);
    }
};
//# sourceMappingURL=getLCP.js.map