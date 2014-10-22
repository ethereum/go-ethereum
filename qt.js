(function() {
    var QtProvider = function() {
        this.handlers = [];
        
        var self = this;
        navigator.qt.onmessage = function (message) {
            self.handlers.forEach(function (handler) {
                handler.call(self, JSON.parse(message));
            });
        }
    };

    QtProvider.prototype.send = function(payload) {
        navigator.qt.postData(JSON.stringify(payload));
    };

    Object.defineProperty(QtProvider.prototype, "onmessage", {
        set: function(handler) {
            this.handlers.push(handler);
        },
    }); 

    if(typeof(web3) !== "undefined" && web3.providers !== undefined) {
        web3.providers.QtProvider = QtProvider;
    }
})();

