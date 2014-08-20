var Filter = function(eth, options) {
	this.callbacks = {};
	this.seed = Math.floor(Math.random() * 1000000);
	this.eth = eth;

	eth.registerFilter(options, this.seed);
};

Filter.prototype.changed = function(callback) {
	var cbseed = Math.floor(Math.random() * 1000000);
	this.eth.registerFilterCallback(this.seed, cbseed);

	var self = this;
	message.connect(function(messages, seed, callbackSeed) {
		if(seed ==  self.seed && callbackSeed == cbseed) {
			callback.call(self, messages);
		}
	});
};

Filter.prototype.uninstall = function() {
	eth.uninstallFilter(this.seed)
}
