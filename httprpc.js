(function () {
    var HttpRpcProvider = function (host) {
        this.handlers = [];
        this.host = host;
    };

    function formatJsonRpcObject(object) {
        return {
            jsonrpc: '2.0',
            method: object.call,
            params: object.args,
            id: object._id
        }
    };

    function formatJsonRpcMessage(message) {    
        var object = JSON.parse(message);
       
        return {
            _id: object.id,
            data: object.result
        };
    };

    HttpRpcProvider.prototype.sendRequest = function (payload, cb) {
        var data = formatJsonRpcObject(payload);

        var request = new XMLHttpRequest();
        request.open("POST", this.host, true);
        request.send(JSON.stringify(data));
        request.onreadystatechange = function () {
            if (request.readyState === 4 && cb) {
                cb(request);
            }
        }
    };

    HttpRpcProvider.prototype.send = function (payload) {
        var self = this;
        this.sendRequest(payload, function (request) {
            self.handlers.forEach(function (handler) {
                handler.call(self, formatJsonRpcMessage(request.responseText));
            });
        });
    };

    HttpRpcProvider.prototype.poll = function (payload, id) {
        var self = this;
        this.sendRequest(payload, function (request) {
            var parsed = JSON.parse(request.responseText);
            if (!parsed.result) {
                return;
            }
            self.handlers.forEach(function (handler) {
                handler.call(self, {_event: payload.call, _id: id, data: parsed.result});
            });
        });
    };

    Object.defineProperty(HttpRpcProvider.prototype, "onmessage", {
        set: function (handler) {
            this.handlers.push(handler);
        }
    });

    if (typeof(web3) !== "undefined" && web3.providers !== undefined) {
        web3.providers.HttpRpcProvider = HttpRpcProvider;
    }
})();

