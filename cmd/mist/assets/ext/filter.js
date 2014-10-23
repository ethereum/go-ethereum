// Copyright (c) 2013-2014, Jeffrey Wilcke. All rights reserved.
//
// This library is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public
// License as published by the Free Software Foundation; either
// version 2.1 of the License, or (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this library; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
// MA 02110-1301  USA

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
