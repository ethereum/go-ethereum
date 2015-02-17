// Copyright (c) 2015, ETHDEV. All rights reserved.
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

// this function is included locally, but you can also include separately via a header definition

console.log("loaded?");

document.onkeydown = function(evt) {
    evt = evt || window.event;
    if (evt.ctrlKey && evt.keyCode == 67) {
    	window.document.execCommand("copy");
        console.log("Ctrl-C");
    } else if (evt.ctrlKey && evt.keyCode == 88) {
    	window.document.execCommand("cut");
        console.log("Ctrl-X");
    } if (evt.ctrlKey && evt.keyCode == 86) {
        console.log("Ctrl-V");
    } if (evt.ctrlKey && evt.keyCode == 90) {
        console.log("Ctrl-Z");
    }
};