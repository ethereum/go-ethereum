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

String.prototype.pad = function(l, r) {
	if (r === undefined) {
		r = l
		if (!(this.substr(0, 2) == "0x" || /^\d+$/.test(this)))
			l = 0
	}
	var ret = this.bin();
	while (ret.length < l)
		ret = "\0" + ret
	while (ret.length < r)
		ret = ret + "\0"
	return ret;
}

String.prototype.unpad = function() {
	var i = this.length;
	while (i && this[i - 1] == "\0")
		--i
	return this.substr(0, i)
}

String.prototype.bin = function() {
	if (this.substr(0, 2) == "0x") {
		bytes = []
		var i = 2;

		// Check if it's odd - pad with a zero if so.
		if (this.length % 2)
			bytes.push(parseInt(this.substr(i++, 1), 16))

		for (; i < this.length - 1; i += 2)
			bytes.push(parseInt(this.substr(i, 2), 16));

		return String.fromCharCode.apply(String, bytes);
	} else if (/^\d+$/.test(this))
		return bigInt(this.substr(0)).toHex().bin()

	// Otherwise we'll return the "String" object instead of an actual string
	return this.substr(0, this.length)
}

String.prototype.unbin = function() {
	var i, l, o = '';
	for(i = 0, l = this.length; i < l; i++) {
		var n = this.charCodeAt(i).toString(16);
		o += n.length < 2 ? '0' + n : n;
	}

	return "0x" + o;
}

String.prototype.dec = function() {
	return bigInt(this.substr(0)).toString()
}

String.prototype.hex = function() {
	return bigInt(this.substr(0)).toHex()
}
