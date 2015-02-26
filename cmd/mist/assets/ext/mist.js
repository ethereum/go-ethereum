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
    // This functions keeps track of keyboard inputs in order to allow copy, paste and other features

    evt = evt || window.event;
    if (evt.ctrlKey && evt.keyCode == 67) {
    	window.document.execCommand("copy");
    } else if (evt.ctrlKey && evt.keyCode == 88) {
    	window.document.execCommand("cut");
    } else if (evt.ctrlKey && evt.keyCode == 86) {
        window.document.execCommand("paste");
    } else if (evt.ctrlKey && evt.keyCode == 90) {
        window.document.execCommand("undo");
    } else if (evt.ctrlKey && evt.shiftKey && evt.keyCode == 90) {
        window.document.execCommand("redo");
    }
};