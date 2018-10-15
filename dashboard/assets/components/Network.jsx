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
import type {Network as NetworkType, PeerEvent} from '../types/content';

// inserter is a state updater function for the main component, which handles the peers.
export const inserter = (sampleLimit: number) => (update: NetworkType, prev: NetworkType) => {
	if (update.peers && update.peers.bundles) {
		prev.peers = update.peers;
	}
	if (Array.isArray(update.diff)) {
		update.diff.forEach((event: PeerEvent) => {
			if (event.removeIP) {
				if (event.removeID && prev.peers.bundles[event.removeIP]) {
					delete prev.peers.bundles[event.removeIP].knownPeers[event.removeID];
				}
				delete prev.peers.bundles[event.removeIP];
				return;
			}
			if (!event.ip) {
				console.error('Peer event without IP', event);
				return;
			}
			if (!prev.peers.bundles[event.ip]) {
				prev.peers.bundles[event.ip] = {
					location:     {},
					knownPeers:   {},
					unknownPeers: [],
				};
			}
			const bundle = prev.peers.bundles[event.ip];
			if (event.location) {
				bundle.location = event.location;
			}
			if (!event.id) {
				bundle.unknownPeers.push({
					connected:    event.connected,
					disconnected: event.disconnected,
				});
				return;
			}
			if (!bundle.knownPeers[event.id]) {
				bundle.knownPeers[event.id] = {
					connected:    [],
					disconnected: [],
					ingress:      [],
					egress:       [],
				};
			}
			const peer = bundle.knownPeers[event.id];
			if (event.connected) {
				peer.connected.push(event.connected);
			}
			if (event.disconnected) {
				peer.disconnected.push(event.disconnected);
			}
			if (Array.isArray(event.ingress) && Array.isArray(event.egress)) {
				if (event.ingress.length !== event.egress.length) {
					console.error('Different traffic sample length', event);
					return;
				}
				if (peer.ingress.length > 0) {
					if (peer.ingress[peer.ingress.length - 1].value < event.ingress[0].value) {
						event.ingress[0].value -= peer.ingress[peer.ingress.length - 1].value;
						event.egress[0].value -= peer.egress[peer.egress.length - 1].value;
					}
				}
				for (let i = 1; i < event.ingress.length; i++) {
					event.ingress[i].value -= event.ingress[i - 1].value;
					event.egress[i].value -= event.egress[i - 1].value;
				}
				peer.ingress.splice(peer.ingress.length, 0, ...event.ingress);
				peer.egress.splice(peer.egress.length, 0, ...event.egress);
				if (peer.ingress.length > sampleLimit) {
					peer.ingress.splice(0, peer.ingress.length - sampleLimit);
				}
				if (peer.egress.length > sampleLimit) {
					peer.egress.splice(0, peer.egress.length - sampleLimit);
				}
				// console.log(event.ingress, prev.peers.bundles[event.ip].knownPeers[event.id].ingress);
			}
		});
	}
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
						<TableCell>Unknown</TableCell>
						<TableCell>Node ID</TableCell>
						<TableCell>Ingress</TableCell>
						<TableCell>Egress</TableCell>
						<TableCell>Connected</TableCell>
						<TableCell>Disconnected</TableCell>
					</TableRow>
				</TableHead>
				<TableBody>
					{Object.entries(this.props.content.peers.bundles).map(([ip, bundle]) => (
						<TableRow key={ip}>
							<TableCell>{ip}</TableCell>
							<TableCell>
								{bundle.location ? (() => {
									const l = bundle.location;
									return `${l.country ? l.country : ''}${l.city ? `/${l.city}` : ''} ${l.latitude} ${l.longitude}`;
								})() : ''}
							</TableCell>
							<TableCell>
								{bundle.unknownPeers && Object.values(bundle.unknownPeers).map(peer => peer.connected && peer.disconnected && `${this.formatTime(peer.connected)}~${this.formatTime(peer.disconnected)}`).join(', ')}
							</TableCell>
							<TableCell>
								{bundle.knownPeers && Object.keys(bundle.knownPeers).map(id => id.substring(0, 10)).join(' ')}
							</TableCell>
							<TableCell>
								{bundle.knownPeers && Object.values(bundle.knownPeers).map(peer => peer.ingress && peer.ingress.map(sample => sample.value).join(' ')).join(', ')}
							</TableCell>
							<TableCell>
								{bundle.knownPeers && Object.values(bundle.knownPeers).map(peer => peer.egress && peer.egress.map(sample => sample.value).join(' ')).join(', ')}
							</TableCell>
							<TableCell>
								{bundle.knownPeers && Object.values(bundle.knownPeers).map(peer => peer.connected && peer.connected.map(time => this.formatTime(time)).join(' ')).join(', ')}
							</TableCell>
							<TableCell>
								{bundle.knownPeers && Object.values(bundle.knownPeers).map(peer => peer.disconnected && peer.disconnected.map(time => this.formatTime(time)).join(' ')).join(', ')}
							</TableCell>
						</TableRow>
					))}
				</TableBody>
			</Table>
		);
	}
}

export default Network;
