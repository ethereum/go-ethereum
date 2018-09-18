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

import Table from '@material-ui/core/Table';
import TableHead from '@material-ui/core/TableHead';
import TableBody from '@material-ui/core/TableBody';
import TableRow from '@material-ui/core/TableRow';
import TableCell from '@material-ui/core/TableCell';
import type {Network as NetworkType, PeerBundle, Peer} from '../types/content';

// inserter is a state updater function for the main component, which handles the peers.
export const inserter = (update: {[string]: PeerBundle}, prev: {[string]: PeerBundle}) => {
	Object.keys(update).forEach((ip) => {
		if (!update[ip]) {
			return;
		}
		if (!prev[ip]) {
			prev[ip] = update[ip];
			return;
		}
		if (update[ip].location) {
			prev[ip].location = update[ip].location;
		}
		if (!update[ip].peers) {
			return;
		}
		Object.entries(update[ip].peers).forEach(([id, u]) => {
			if (!prev[ip].peers[id]) {
				prev[ip].peers[id] = u;
				return;
			}
			const p: Peer = prev[ip].peers[id];
			if (u.connected) {
				if (!Array.isArray(p.connected)) {
					p.connected = [];
				}
				p.connected = [...p.connected, ...u.connected];
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
			prev[ip].peers[id] = p;
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

type State = {};

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
						<TableCell>Location</TableCell>
						<TableCell>Peer ID</TableCell>
						<TableCell>Ingress</TableCell>
						<TableCell>Egress</TableCell>
						<TableCell>Connected</TableCell>
						<TableCell>Disconnected</TableCell>
					</TableRow>
				</TableHead>
				<TableBody>
					{Object.entries(this.props.content.peerBundles).map(([ip, bundle]) => { console.log(ip, bundle); return (
						<TableRow key={ip}>
							<TableCell>{ip}</TableCell>
							<TableCell>
								{bundle.location ? (() => {
									const l = bundle.location;
									return `${l.country ? l.country : ''}${l.city ? `/${l.city}` : ''} ${l.latitude} ${l.longitude}`;
								})() : ''}
							</TableCell>
							<TableCell>
								{bundle.peers && Object.keys(bundle.peers).map(id => id.substring(0, 10)).join(' ')}
							</TableCell>
							<TableCell>
								{bundle.peers && Object.values(bundle.peers).map(peer => peer.ingress && peer.ingress.map(sample => sample.value).join(' ')).join(', ')}
							</TableCell>
							<TableCell>
								{bundle.peers && Object.values(bundle.peers).map(peer => peer.egress && peer.egress.map(sample => sample.value).join(' ')).join(', ')}
							</TableCell>
							<TableCell>
								{bundle.peers && Object.values(bundle.peers).map(peer => peer.connected && peer.connected.map(time => this.formatTime(time)).join(' ')).join(', ')}
							</TableCell>
							<TableCell>
								{bundle.peers && Object.values(bundle.peers).map(peer => peer.disconnected && peer.disconnected.map(time => this.formatTime(time)).join(' ')).join(', ')}
							</TableCell>
						</TableRow>
					)})}
				</TableBody>
			</Table>
		);
	}
}

export default Network;
