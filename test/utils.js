var assert = require('assert');

var methodExists = function (object, method) {
    assert.equal('function', typeof object[method], 'method ' + method + ' is not implemented');
};

var propertyExists = function (object, property) {
    assert.equal('object', typeof object[property], 'property ' + property + ' is not implemented');
};

module.exports = {
    methodExists: methodExists,
    propertyExists: propertyExists
};

