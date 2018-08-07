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

import Table, {TableHead, TableBody, TableRow, TableCell} from 'material-ui/Table';
import type {Network as NetworkType, Peer} from '../types/content';

// inserter is a state updater function for the main component, which inserts the new log chunk into the chunk array.
// limit is the maximum length of the chunk array, used in order to prevent the browser from OOM.
export const inserter = (update: {[string]: {[string]: Peer}}, prev: {[string]: {[string]: Peer}}) => {
	Object.keys(update).forEach((ip) => {
		if (!prev[ip]) {
			prev[ip] = update[ip];
			return;
		}
		if (!update[ip]) {
			return;
		}
		Object.keys(update[ip]).forEach((id) => {
			if (!prev[ip][id]) {
				prev[ip][id] = update[ip][id];
				return;
			}
			const u: Peer = update[ip][id];
			const p: Peer = prev[ip][id];
			if (u.connected) {
				if (!Array.isArray(p.connected)) {
					p.connected = [];
				}
				p.connected = [...p.connected, ...u.connected];
			}
			if (u.handshake) {
				if (!Array.isArray(p.handshake)) {
					p.handshake = [];
				}
				p.handshake = [...p.handshake, ...u.handshake];
			}
			if (u.disconnected) {
				if (!Array.isArray(p.disconnected)) {
					p.disconnected = [];
				}
				p.disconnected = [...p.disconnected, ...u.disconnected];
			}
			if (Array.isArray(u.ingress)) {
				if (!Array.isArray(p.ingress)) {
					p.ingress = [];
				}
				p.ingress = [...p.ingress, ...u.ingress].slice(-200);
			}
			if (Array.isArray(u.egress)) {
				if (!Array.isArray(p.egress)) {
					p.egress = [];
				}
				p.egress = [...p.egress, ...u.egress].slice(-200);
			}
			prev[ip][id] = p;
		});
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
	formatTime = (t) => {
		const time = new Date(t);
		if (isNaN(time)) {
			return '';
		}
		const month = `0${time.getMonth() + 1}`.slice(-2);
		const date = `0${time.getDate()}`.slice(-2);
		const hours = `0${time.getHours()}`.slice(-2);
		const minutes = `0${time.getMinutes()}`.slice(-2);
		const seconds = `0${time.getSeconds()}`.slice(-2);
		return `${month}/${date}/${hours}:${minutes}:${seconds}`;
	};

	render() {
		return (
			<Table>
				<TableHead>
					<TableRow>
						<TableCell>IP</TableCell>
						<TableCell>Peer ID</TableCell>
						<TableCell>Location</TableCell>
						<TableCell>Ingress</TableCell>
						<TableCell>Egress</TableCell>
						<TableCell>Connected</TableCell>
						<TableCell>Handshake</TableCell>
						<TableCell>Disconnected</TableCell>
					</TableRow>
				</TableHead>
				<TableBody>
					{Object.entries(this.props.content.peers).map(([ip, peers]) => (
						<TableRow key={ip}>
							<TableCell>{ip}</TableCell>
							<TableCell>
								{Object.keys(peers).map(id => id.substring(0, 10)).join(' ')}
							</TableCell>
							<TableCell>
								{(() => {
									const k = Object.keys(peers)[0];
									return k && peers[k].location ? (() => {
										const l = peers[k].location;
										return `${l.country}${l.city ? `/${l.city}` : ''} ${l.latitude} ${l.longitude}`;
									})() : '';
								})()}
							</TableCell>
							<TableCell>
								{Object.keys(peers).map((id) => peers[id].ingress && peers[id].ingress.map(sample => sample.value).join(' ')).join(', ')}
							</TableCell>
							<TableCell>
								{Object.keys(peers).map((id) => peers[id].egress && peers[id].egress.map(sample => sample.value).join(' ')).join(', ')}
							</TableCell>
							<TableCell>
								{Object.keys(peers).map((id) => peers[id].connected && peers[id].connected.map(time => this.formatTime(time)).join(' ')).join(', ')}
							</TableCell>
							<TableCell>
								{Object.keys(peers).map((id) => peers[id].handshake && peers[id].handshake.map(time => this.formatTime(time)).join(' ')).join(', ')}
							</TableCell>
							<TableCell>
								{Object.keys(peers).map((id) => peers[id].disconnected && peers[id].disconnected.map(time => this.formatTime(time)).join(' ')).join(', ')}
							</TableCell>
						</TableRow>
					))}
				</TableBody>
			</Table>
		);
	}
}

export default Network;
