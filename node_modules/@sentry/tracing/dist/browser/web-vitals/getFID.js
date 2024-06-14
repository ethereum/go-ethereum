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
exports.getFID = function (onReport) {
    var metric = initMetric_1.initMetric('FID');
    var firstHidden = getFirstHidden_1.getFirstHidden();
    var entryHandler = function (entry) {
        // Only report if the page wasn't hidden prior to the first input.
        if (entry.startTime < firstHidden.timeStamp) {
            metric.value = entry.processingStart - entry.startTime;
            metric.entries.push(entry);
            metric.isFinal = true;
            report();
        }
    };
    var po = observe_1.observe('first-input', entryHandler);
    var report = bindReporter_1.bindReporter(onReport, metric, po);
    if (po) {
        onHidden_1.onHidden(function () {
            po.takeRecords().map(entryHandler);
            po.disconnect();
        }, true);
    }
    else {
        if (window.perfMetrics && window.perfMetrics.onFirstInputDelay) {
            window.perfMetrics.onFirstInputDelay(function (value, event) {
                // Only report if the page wasn't hidden prior to the first input.
                if (event.timeStamp < firstHidden.timeStamp) {
                    metric.value = value;
                    metric.isFinal = true;
                    metric.entries = [
                        {
                            entryType: 'first-input',
                            name: event.type,
                            target: event.target,
                            cancelable: event.cancelable,
                            startTime: event.timeStamp,
                            processingStart: event.timeStamp + value,
                        },
                    ];
                    report();
                }
            });
        }
    }
};
//# sourceMappingURL=getFID.js.map