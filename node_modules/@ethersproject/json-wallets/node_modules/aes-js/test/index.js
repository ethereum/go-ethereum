
function populateTests(tests) {
    for (var key in tests) {
        module.exports[key] = tests[key];
    }
}

populateTests(require('./test-aes.js'));
populateTests(require('./test-counter.js'));
populateTests(require('./test-buffer.js'));
populateTests(require('./test-errors.js'));
populateTests(require('./test-padding.js'));
