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
import AreaChart from 'recharts/es6/chart/AreaChart';
import Tooltip from 'recharts/es6/component/Tooltip';
import Area from 'recharts/es6/cartesian/Area';
import CustomTooltip, {bytePlotter, multiplier} from 'CustomTooltip';
import type {Network as NetworkType, PeerEvent} from '../types/content';

// inserter is a state updater function for the main component, which handles the peers.
export const inserter = (sampleLimit: number) => (update: NetworkType, prev: NetworkType) => {
	if (update.peers && update.peers.bundles) {
		prev.peers = update.peers;
	}
	if (Array.isArray(update.diff)) {
		update.diff.forEach((event: PeerEvent) => {
			if (!event.ip) {
				console.error('Peer event without IP', event);
				return;
			}
			switch (event.remove) {
			case 'bundle':
				delete prev.peers.bundles[event.ip];
				return;
			case 'known': {
				if (!event.id) {
					console.error('Remove known peer event without ID', event.ip);
					return;
				}
				const bundle = prev.peers.bundles[event.ip];
				if (!bundle || !bundle.knownPeers || !bundle.knownPeers[event.id]) {
					console.error('No known peer to remove', event.ip, event.id);
					return;
				}
				delete bundle.knownPeers[event.id];
				return;
			}
			case 'unknown': {
				const bundle = prev.peers.bundles[event.ip];
				if (!bundle || !Array.isArray(bundle.unknownPeers) || bundle.unknownPeers.length < 1) {
					console.error('No unknown peer to remove', event.ip);
					return;
				}
				bundle.unknownPeers.splice(0, 1);
				return;
			}
			}
			if (!prev.peers.bundles[event.ip]) {
				prev.peers.bundles[event.ip] = {
					location: {
						country:   '',
						city:      '',
						latitude:  0,
						longitude: 0,
					},
					knownPeers:   {},
					unknownPeers: [],
				};
			}
			const bundle = prev.peers.bundles[event.ip];
			if (event.location) {
				bundle.location = event.location;
				return;
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
				peer.ingress.splice(peer.ingress.length, 0, ...event.ingress);
				peer.egress.splice(peer.egress.length, 0, ...event.egress);
				if (peer.ingress.length > sampleLimit) {
					peer.ingress.splice(0, peer.ingress.length - sampleLimit);
				}
				if (peer.egress.length > sampleLimit) {
					peer.egress.splice(0, peer.egress.length - sampleLimit);
				}
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
			<div>
				<Table>
					<TableHead>
						<TableRow>
							<TableCell>IP</TableCell>
							<TableCell>Location</TableCell>
							<TableCell>Node ID</TableCell>
							<TableCell>Traffic</TableCell>
							<TableCell>Connected</TableCell>
							<TableCell>Disconnected</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{Object.entries(this.props.content.peers.bundles).map(([ip, bundle]) => {
							if (!bundle.knownPeers || Object.keys(bundle.knownPeers).length < 1) {
								return null;
							}
							return (
								<TableRow key={`known${ip}`}>
									<TableCell>{ip}</TableCell>
									<TableCell>
										{bundle.location ? (() => {
											const l = bundle.location;
											return `${l.country ? l.country : ''}${l.city ? `/${l.city}` : ''} ${l.latitude} ${l.longitude}`;
										})() : ''}
									</TableCell>
									<TableCell>
										{Object.keys(bundle.knownPeers).map(id => id.substring(0, 10)).join(' ')}
									</TableCell>
									<TableCell>
										{Object.values(bundle.knownPeers).map(({ingress, egress}) => (
											<div>
												<AreaChart
													width={300} height={50}
													syncId={'footerSyncId'}
													data={egress.map(({value}) => ({egress: value || 0}))}
													margin={{top: 5, right: 5, bottom: 0, left: 5}}
												>
													<Tooltip cursor={false} content={<CustomTooltip tooltip={bytePlotter('Download')} />} />
													<Area isAnimationActive={false} type='monotone' dataKey='egress' stroke='#8884d8' fill='#8884d8' />
												</AreaChart>
												<AreaChart
													width={300} height={50}
													syncId={'footerSyncId'}
													data={ingress.map(({value}) => ({ingress: -value || 0}))}
													margin={{top: 0, right: 5, bottom: 5, left: 5}}
												>
													<Tooltip cursor={false} content={<CustomTooltip tooltip={bytePlotter('Upload', multiplier(-1))} />} />
													<Area isAnimationActive={false} type='monotone' dataKey='ingress' stroke='#82ca9d' fill='#82ca9d' />
												</AreaChart>
											</div>
										))}
									</TableCell>
									<TableCell>
										{Object.values(bundle.knownPeers).map(peer => peer.connected && peer.connected.map(time => this.formatTime(time)).join(' ')).join(', ')}
									</TableCell>
									<TableCell>
										{Object.values(bundle.knownPeers).map(peer => peer.disconnected && peer.disconnected.map(time => this.formatTime(time)).join(' ')).join(', ')}
									</TableCell>
								</TableRow>
							);
						})}
					</TableBody>
				</Table>
				<Table>
					<TableHead>
						<TableRow>
							<TableCell>IP</TableCell>
							<TableCell>Location</TableCell>
							<TableCell>Connected</TableCell>
							<TableCell>Disconnected</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{Object.entries(this.props.content.peers.bundles).map(([ip, bundle]) => {
							if (!bundle.unknownPeers || bundle.unknownPeers.length < 1) {
								return null;
							}
							return (
								<TableRow key={`unknown${ip}`}>
									<TableCell>{ip}</TableCell>
									<TableCell>
										{bundle.location ? (() => {
											const l = bundle.location;
											return `${l.country ? l.country : ''}${l.city ? `/${l.city}` : ''} ${l.latitude} ${l.longitude}`;
										})() : ''}
									</TableCell>
									<TableCell>
										{Object.values(bundle.unknownPeers).map(peer => peer.connected && `${this.formatTime(peer.connected)}`).join(', ')}
									</TableCell>
									<TableCell>
										{Object.values(bundle.unknownPeers).map(peer => peer.disconnected && `${this.formatTime(peer.disconnected)}`).join(', ')}
									</TableCell>
								</TableRow>
							);
						})}
					</TableBody>
				</Table>
			</div>);
	}
}

export default Network;
