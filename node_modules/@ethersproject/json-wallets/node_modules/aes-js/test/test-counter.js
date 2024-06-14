var nodeunit = require('nodeunit');

var aes = require('../index');

function bufferEquals(a, b) {
    if (a.length != b.length) { return false; }
    for (var i = 0; i < a.length; i++) {
        if (a[i] != b[i]) { return false; }
    }
    return true;
}

function makeTest (options) {
    return function(test) {
        var result = new Buffer(options.incrementResult, 'hex');

        if (options.hasOwnProperty('nullish')) {
            var counter = new aes.Counter(options.nullish);
            counter.increment();
            test.ok(bufferEquals(counter._counter, result), "counter failed to initialize with a nullish thing")
        }

        if (options.hasOwnProperty('number')) {

            var counter = new aes.Counter(options.number);
            counter.increment();
            test.ok(bufferEquals(counter._counter, result), "counter failed to initialize with a number")

            counter.setValue(options.number);
            counter.increment();
            test.ok(bufferEquals(counter._counter, result), "counter failed to reset to a number")

            counter = new aes.Counter();
            counter.setValue(options.number);
            counter.increment();
            test.ok(bufferEquals(counter._counter, result), "counter failed to reset to a number")
        }

        if (options.bytes) {
            var bytes = new Buffer(options.bytes, 'hex');

            var counter = new aes.Counter(bytes);
            counter.increment();
            test.ok(bufferEquals(counter._counter, result), "counter failed to initialize with bytes")

            counter.setBytes(bytes);
            counter.increment();
            test.ok(bufferEquals(counter._counter, result), "counter failed to reset with bytes")

            counter = new aes.Counter();
            counter.setBytes(bytes);
            counter.increment();
            test.ok(bufferEquals(counter._counter, result), "counter failed to reset with bytes")
        }

        test.done();
    };
}

module.exports = {
    'test-counter-nullish-null': makeTest({nullish: null, incrementResult: "00000000000000000000000000000002"}),
    'test-counter-nullish-undefined': makeTest({nullish: undefined, incrementResult: "00000000000000000000000000000002"}),
    'test-counter-number-0': makeTest({number: 0, incrementResult: "00000000000000000000000000000001"}),
    'test-counter-number-1': makeTest({number: 1, incrementResult: "00000000000000000000000000000002"}),
    'test-counter-number-254': makeTest({number: 254, incrementResult: "000000000000000000000000000000ff"}),
    'test-counter-number-255': makeTest({number: 255, incrementResult: "00000000000000000000000000000100"}),
    'test-counter-number-256': makeTest({number: 256, incrementResult: "00000000000000000000000000000101"}),
    'test-counter-bytes-0000': makeTest({bytes: "00000000000000000000000000000000", incrementResult: "00000000000000000000000000000001"}),
    'test-counter-bytes-00ff': makeTest({bytes: "000000000000000000000000000000ff", incrementResult: "00000000000000000000000000000100"}),
    'test-counter-bytes-ffff': makeTest({bytes: "ffffffffffffffffffffffffffffffff", incrementResult: "00000000000000000000000000000000"}),
    'test-counter-bytes-dead': makeTest({bytes: "deadbeefdeadbeefdeadbeefdeadbeef", incrementResult: "deadbeefdeadbeefdeadbeefdeadbef0"}),
};

