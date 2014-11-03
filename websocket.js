(function() {
    var WebSocketProvider = function(host) {
        // onmessage handlers
        this.handlers = [];
        // queue will be filled with messages if send is invoked before the ws is ready
        this.queued = [];
        this.ready = false;

        this.ws = new WebSocket(host);

        var self = this;
        this.ws.onmessage = function(event) {
            for(var i = 0; i < self.handlers.length; i++) {
                self.handlers[i].call(self, JSON.parse(event.data), event);
            }
        };

        this.ws.onopen = function() {
            self.ready = true;

            for(var i = 0; i < self.queued.length; i++) {
                // Resend
                self.send(self.queued[i]);
            }
        };
    };
    WebSocketProvider.prototype.send = function(payload) {
        if(this.ready) {
            var data = JSON.stringify(payload);

            this.ws.send(data);
        } else {
            this.queued.push(payload);
        }
    };

    WebSocketProvider.prototype.onMessage = function(handler) {
        this.handlers.push(handler);
    };

    WebSocketProvider.prototype.unload = function() {
        this.ws.close();
    };
    Object.defineProperty(WebSocketProvider.prototype, "onmessage", {
        set: function(provider) { this.onMessage(provider); }
    });

    if(typeof(web3) !== "undefined" && web3.providers !== undefined) {
        web3.providers.WebSocketProvider = WebSocketProvider;
    }
})();
