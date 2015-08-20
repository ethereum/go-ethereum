/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file shh.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var Method = require('../method');
var formatters = require('../formatters');

var post = new Method({
    name: 'post', 
    call: 'shh_post', 
    params: 1,
    inputFormatter: [formatters.inputPostFormatter]
});

var newIdentity = new Method({
    name: 'newIdentity',
    call: 'shh_newIdentity',
    params: 0
});

var hasIdentity = new Method({
    name: 'hasIdentity',
    call: 'shh_hasIdentity',
    params: 1
});

var newGroup = new Method({
    name: 'newGroup',
    call: 'shh_newGroup',
    params: 0
});

var addToGroup = new Method({
    name: 'addToGroup',
    call: 'shh_addToGroup',
    params: 0
});

var methods = [
    post,
    newIdentity,
    hasIdentity,
    newGroup,
    addToGroup
];

module.exports = {
    methods: methods
};

