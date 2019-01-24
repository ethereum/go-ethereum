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
import Grid from '@material-ui/core/Grid/Grid';
import Typography from '@material-ui/core/Typography';
import AreaChart from 'recharts/es6/chart/AreaChart';
import Tooltip from 'recharts/es6/component/Tooltip';
import Area from 'recharts/es6/cartesian/Area';
import {Icon as FontAwesome} from 'react-fa';

import CustomTooltip, {bytePlotter, multiplier} from 'CustomTooltip';
import type {Network as NetworkType, PeerEvent} from '../types/content';
import {styles as commonStyles} from '../common';

// inserter is a state updater function for the main component, which handles the peers.
export const inserter = (sampleLimit: number) => (update: NetworkType, prev: NetworkType) => {
	// The first message contains the metered peer history,
	// which has a valid peer state JSON format per se.
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
			case 'bundle': {
				delete prev.peers.bundles[event.ip];
				return;
			}
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
			case 'attempt': {
				const bundle = prev.peers.bundles[event.ip];
				if (!bundle || !Array.isArray(bundle.attempts) || bundle.attempts.length < 1) {
					console.error('No unknown peer to remove', event.ip);
					return;
				}
				bundle.attempts.splice(0, 1);
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
					knownPeers: {},
					attempts:   [],
				};
			}
			const bundle = prev.peers.bundles[event.ip];
			if (event.location) {
				bundle.location = event.location;
				return;
			}
			if (!event.id) {
				if (!bundle.attempts) {
					bundle.attempts = [];
				}
				bundle.attempts.push({
					connected:    event.connected,
					disconnected: event.disconnected,
				});
				return;
			}
			if (!bundle.knownPeers) {
				bundle.knownPeers = {};
			}
			if (!bundle.knownPeers[event.id]) {
				bundle.knownPeers[event.id] = {
					connected:    [],
					disconnected: [],
					ingress:      [],
					egress:       [],
					active:       false,
				};
			}
			const peer = bundle.knownPeers[event.id];
			if (event.connected) {
				if (!peer.connected) {
					console.warn('peer.connected should exist');
					peer.connected = [];
				}
				peer.connected.push(event.connected);
			}
			if (event.disconnected) {
				if (!peer.disconnected) {
					console.warn('peer.disconnected should exist');
					peer.disconnected = [];
				}
				peer.disconnected.push(event.disconnected);
			}
			switch (event.activity) {
			case 'active':
				peer.active = true;
				break;
			case 'inactive':
				peer.active = false;
				break;
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
const styles = {
	tableHead: {
		height: 'auto',
	},
	tableRow: {
		height: 'auto',
	},
	tableCell: {
		paddingTop:    0,
		paddingRight:  5,
		paddingBottom: 0,
		paddingLeft:   5,
		border:        'none',
	},
};

export type Props = {
    container:    Object,
    content:      NetworkType,
    shouldUpdate: Object,
};

type State = {};

// Network renders the network page.
class Network extends Component<Props, State> {
	formatTime = (t: string) => {
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

	copyToClipboard = (id) => (event) => {
		event.preventDefault();
		navigator.clipboard.writeText(id).then(() => {}, () => {
			console.error("Failed to copy node id", id);
		});
	};

	// TODO (kurkomisi): add single tooltip and move it to the mouse position on copy button click.
	// Tried with TooltipTrigger components for each button, but it seems to be a big load.
	peerTableRow = (ip, id, bundle, peer) => (
		<TableRow key={`known_${ip}_${id}`} style={styles.tableRow}>
			<TableCell style={styles.tableCell}>
				<FontAwesome name='circle' style={{color: peer.active ? 'green' : 'red'}} />
			</TableCell>
			<TableCell style={{fontFamily: 'monospace', ...styles.tableCell}}>
				{id.substring(0, 10) + ' '}
				<FontAwesome name='copy' style={commonStyles.light} onClick={this.copyToClipboard(id)} />
			</TableCell>
			<TableCell style={styles.tableCell}>
				{bundle.location ? (() => {
					const l = bundle.location;
					return `${l.country ? l.country : ''}${l.city ? `/${l.city}` : ''}`;
				})() : ''}
			</TableCell>
			<TableCell style={styles.tableCell}>
				<AreaChart
					width={200} height={18}
					syncId={'footerSyncId'}
					data={peer.ingress.map(({value}) => ({ingress: value || 0}))}
					margin={{top: 5, right: 5, bottom: 0, left: 5}}
				>
					<Tooltip cursor={false} content={<CustomTooltip tooltip={bytePlotter('Download')} />} />
					<Area isAnimationActive={false} type='monotone' dataKey='ingress' stroke='#8884d8' fill='#8884d8' />
				</AreaChart>
				<AreaChart
					width={200} height={18}
					syncId={'footerSyncId'}
					data={peer.egress.map(({value}) => ({egress: -value || 0}))}
					margin={{top: 0, right: 5, bottom: 5, left: 5}}
				>
					<Tooltip cursor={false} content={<CustomTooltip tooltip={bytePlotter('Upload', multiplier(-1))} />} />
					<Area isAnimationActive={false} type='monotone' dataKey='egress' stroke='#82ca9d' fill='#82ca9d' />
				</AreaChart>
			</TableCell>
		</TableRow>
	);

	render() {
		return (
			<Grid container direction='row' justify='space-between'>
				<Grid item>
					<Typography variant='subtitle1' gutterBottom>
						Known peers
					</Typography>
					<Table>
						<TableHead style={styles.tableHead}>
							<TableRow style={styles.tableRow}>
								<TableCell style={styles.tableCell} />
								<TableCell style={styles.tableCell}>Node ID</TableCell>
								<TableCell style={styles.tableCell}>Location</TableCell>
								<TableCell style={styles.tableCell}>Traffic</TableCell>
							</TableRow>
						</TableHead>
						<TableBody>
							{Object.entries(this.props.content.peers.bundles).map(([ip, bundle]) => {
								if (!bundle.knownPeers || Object.keys(bundle.knownPeers).length < 1) {
									return null;
								}
								return Object.entries(bundle.knownPeers).map(([id, peer]) => {
									if (peer.active === false) {
										return null;
									}
									return this.peerTableRow(ip, id, bundle, peer);
								});
							})}
						</TableBody>
						<TableBody>
							{Object.entries(this.props.content.peers.bundles).map(([ip, bundle]) => {
								if (!bundle.knownPeers || Object.keys(bundle.knownPeers).length < 1) {
									return null;
								}
								return Object.entries(bundle.knownPeers).map(([id, peer]) => {
									if (peer.active === true) {
										return null;
									}
									return this.peerTableRow(ip, id, bundle, peer);
								});
							})}
						</TableBody>
					</Table>
				</Grid>
				<Grid item>
					<Typography variant='subtitle1' gutterBottom>
						Connection attempts
					</Typography>
					<Table>
						<TableHead style={styles.tableHead}>
							<TableRow style={styles.tableRow}>
								<TableCell style={styles.tableCell}>IP</TableCell>
								<TableCell style={styles.tableCell}>Location</TableCell>
								<TableCell style={styles.tableCell}>Nr</TableCell>
							</TableRow>
						</TableHead>
						<TableBody>
							{Object.entries(this.props.content.peers.bundles).map(([ip, bundle]) => {
								if (!bundle.attempts || bundle.attempts.length < 1) {
									return null;
								}
								return (
									<TableRow key={`attempt_${ip}`} style={styles.tableRow}>
										<TableCell style={styles.tableCell}>{ip}</TableCell>
										<TableCell style={styles.tableCell}>
											{bundle.location ? (() => {
												const l = bundle.location;
												return `${l.country ? l.country : ''}${l.city ? `/${l.city}` : ''}`;
											})() : ''}
										</TableCell>
										<TableCell style={styles.tableCell}>
											{Object.values(bundle.attempts).length}
										</TableCell>
									</TableRow>
								);
							})}
						</TableBody>
					</Table>
				</Grid>
			</Grid>
		);
	}
}

export default Network;
