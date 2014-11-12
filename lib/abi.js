
var findIndex = function (array, callback) {
    var end = false;
    var i = 0;
    for (; i < array.length && !end; i++) {
        end = callback(array[i]);
    }
    return end ? i - 1 : -1;
};

var padLeft = function (number, n) {
    return (new Array(n - number.toString.length + 1)).join("0") + number;
};

var setupTypes = function () {
    var prefixedType = function (prefix) {
        return function (type, value) {
            var expected = prefix;
            if (type.indexOf(expected) !== 0) {
                return false;
            }

            var padding = parseInt(type.slice(expected.length)) / 8;
            return padLeft(value, padding);
        };
    };

    var namedType = function (name, padding) {
        return function (type, value) {
            if (type !== name) {
                return false; 
            }

            return padLeft(value, padding);
        };
    };

    return [
        prefixedType('uint'),
        prefixedType('int'),
        namedType('address', 20),
        namedType('bool', 1),
    ];
};

var types = setupTypes();

var toBytes = function (json, methodName, params) {
    var bytes = "";
    var index = findIndex(json, function (method) {
        return method.name === methodName;
    });

    if (index === -1) {
        return;
    }

    bytes = bytes + index + 'x';
    var method = json[index];
    
    for (var i = 0; i < method.inputs.length; i++) {
        var found = false;
        for (var j = 0; j < types.length && !found; j++) {
            found = types[j](method.inputs[i].type, params[i]);
        }
        if (!found) {
            console.error('unsupported json type: ' + method.inputs[i].type);
        }
        bytes += found;
    }
    return bytes;
};

module.exports = {
    toBytes: toBytes
};

