// @flow

// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

import React, {Component} from 'react';

import Table, {TableBody, TableHeader, TableHeaderColumn, TableRow, TableCell} from 'material-ui/Table';
import type {Network as NetworkType, Peer} from '../types/content';

// inserter is a state updater function for the main component, which inserts the new log chunk into the chunk array.
// limit is the maximum length of the chunk array, used in order to prevent the browser from OOM.
export const inserter = (update: {[number]: Peer}, prev: {[number]: Peer}) => {
	Object.keys(update).forEach((k) => {
		if (!prev[k]) {
			prev[k] = update[k];
			return;
		}
		const u: Peer = update[k];
		const p: Peer = prev[k];
		if (u.id) {
			p.id = u.id;
		}
		if (u.ip) {
			p.ip = u.ip;
		}
		if (u.lifecycle) {
			if (u.lifecycle.handshake) {
				p.lifecycle.handshake = u.lifecycle.handshake;
			}
			if (u.lifecycle.disconnected) {
				p.lifecycle.disconnected = u.lifecycle.disconnected;
			}
		}
		p.ingress = [...p.ingress, ...u.ingress].slice(-200);
		p.egress = [...p.egress, ...u.egress].slice(-200);
		prev[k] = p;
	});
	return prev;
};

// styles contains the constant styles of the component.
const styles = {};

export type Props = {
    container:    Object,
    content:      NetworkType,
    shouldUpdate: Object,
};

// Network renders the network page.
class Network extends Component<Props, State> {
	render() {
		return (
			<Table>
				<TableBody>
					{Object.entries(this.props.content.peers).map(([k, v]) => (
						<TableRow>
							<TableCell>{k}</TableCell>
							<TableCell>{v.id ? v.id.substring(0, 6) : ''}</TableCell>
							<TableCell>{v.ip}</TableCell>
							<TableCell>{v.ingress.value}</TableCell>
							<TableCell>{v.egress.value}</TableCell>
							<TableCell>{JSON.stringify(v.location)}</TableCell>
							<TableCell>{JSON.stringify(v.lifecycle)}</TableCell>
						</TableRow>
					))}
				</TableBody>
			</Table>
		);
	}
}

export default Network;
