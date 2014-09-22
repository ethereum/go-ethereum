var ethx = {
    prototype: Object,

    watch: function(options) {
        return new Filter(options);
    },

    note: function() {
        var args = Array.prototype.slice.call(arguments, 0);
        var o = []
        for(var i = 0; i < args.length; i++) {
            o.push(args[i].toString())
        }

        eth.notef(o);
    },
};

var Filter = function(options) {
	this.callbacks = [];
	this.options = options;

	if(options === "chain") {
		this.id = eth.newFilterString(options);
	} else if(typeof options === "object") {
		this.id = eth.newFilter(options);
	}
};

Filter.prototype.changed = function(callback) {
    this.callbacks.push(callback);

	var self = this;
	messages.connect(function(messages, id) {
		if(id ==  self.id) {
			for(var i = 0; i < self.callbacks.length; i++) {
				self.callbacks[i].call(self, messages);
			}
		}
	});
};

Filter.prototype.uninstall = function() {
	eth.uninstallFilter(this.id)
}

Filter.prototype.messages = function() {
	return eth.messages(this.id)
}
