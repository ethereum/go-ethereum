
var abi = require('./abi');

var implementationOfEvent = function (event, address, signature) {
    
    return function (options) {
        var o = options || {};
        o.address = o.address || address;
        o.topics = o.topics || [];
        o.topics.push(signature);
        return o;
    };
};

module.exports = implementationOfEvent;

