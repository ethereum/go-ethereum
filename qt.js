(function() {
    var QtProvider = function() {};
    QtProvider.prototype.send = function(payload) {
        navigator.qt.postData(JSON.stringify(payload));
    };
    Object.defineProperty(QtProvider.prototype, "onmessage", {
        set: function(handler) {
            navigator.qt.onmessage = handler;
        },
    }); 

    if(typeof(web3) !== "undefined" && web3.providers !== undefined) {
        web3.QtProvider = QtProvider;
    }
})();
