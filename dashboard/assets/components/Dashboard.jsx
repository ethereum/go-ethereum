// @flow

// Copyright 2017 The go-ethereum Authors
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

import List, {ListItem} from 'material-ui/List';
import withStyles from 'material-ui/styles/withStyles';

import Header from './Header';
import Body from './Body';
import {MENU} from '../common';
import type {Content, Record, Chunk} from '../types/content';

// deepUpdate updates an object corresponding to the given update data, which has
// the shape of the same structure as the original object. updater also has the same
// structure, except that it contains functions where the original data needs to be
// updated. These functions are used to handle the update.
//
// Since the messages have the same shape as the state content, this approach allows
// the generalization of the message handling. The only necessary thing is to set a
// handler function for every path of the state in order to maximize the flexibility
// of the update.
const deepUpdate = (updater: Object, update: Object, prev: Object): $Shape<Content> => {
	if (typeof update === 'undefined') {
		// TODO (kurkomisi): originally this was deep copy, investigate it.
		return prev;
	}
	if (typeof updater === 'function') {
		return updater(update, prev);
	}
	const updated = {};
	Object.keys(prev).forEach((key) => {
		updated[key] = deepUpdate(updater[key], update[key], prev[key]);
	});

	return updated;
};

// shouldUpdate returns the structure of a message. It is used to prevent unnecessary render
// method triggerings. In the affected component's shouldComponentUpdate method it can be checked
// whether the involved data was changed or not by checking the message structure.
//
// We could return the message itself too, but it's safer not to give access to it.
const shouldUpdate = (updater: Object, msg: Object) => {
	const su = {};
	Object.keys(msg).forEach((key) => {
		su[key] = typeof updater[key] !== 'function' ? shouldUpdate(updater[key], msg[key]) : true;
	});

	return su;
};

// replacer is a state updater function, which replaces the original data.
const replacer = <T>(update: T) => update;

// appender is a state updater function, which appends the update data to the
// existing data. limit defines the maximum allowed size of the created array,
// mapper maps the update data.
const appender = <T>(limit: number, mapper = replacer) => (update: Array<T>, prev: Array<T>) => [
	...prev,
	...update.map(sample => mapper(sample)),
].slice(-limit);

// fieldPadding is a global map with maximum field value lengths seen until now
// to allow padding log contexts in a bit smarter way.
const fieldPadding = new Map();

// createLogChunk creates an HTML formatted object, which displays the given array similarly to
// the server side terminal.
const createLogChunk = (arr: Array<Record>) => {
	let content = '';
	arr.forEach((record) => {
		let {t, lvl, msg, ctx} = record;
		let color = '#ce3c23';
		switch (lvl) {
		case 'trace':
		case 'trce':
			lvl = 'TRACE';
			color = '#3465a4';
			break;
		case 'debug':
		case 'dbug':
			lvl = 'DEBUG';
			color = '#3d989b';
			break;
		case 'info':
			lvl = 'INFO&nbsp;';
			color = '#4c8f0f';
			break;
		case 'warn':
			lvl = 'WARN&nbsp;';
			color = '#b79a22';
			break;
		case 'error':
		case 'eror':
			lvl = 'ERROR';
			color = '#754b70';
			break;
		case 'crit':
			lvl = 'CRIT&nbsp;';
			color = '#ce3c23';
			break;
		default:
			lvl = '';
		}
		if (lvl === '' || typeof t !== 'string' || t.length < 19 || typeof msg !== 'string' || !Array.isArray(ctx)) {
			content += `<span style="color:${color}">Invalid log record</span><br />`;
			return;
		}
		if (ctx.length > 0) {
			msg += '&nbsp;'.repeat(Math.max(40 - msg.length, 0));
		}
		// Time format: 2006-01-02T15:04:05-0700 -> 01-02|15:04:05
		content += `<span style="color:${color}">${lvl}</span>[${t.substr(5, 5)}|${t.substr(11, 8)}] ${msg}`;

		for (let i = 0; i < ctx.length; i += 2) {
			const key = ctx[i];
			const value = ctx[i + 1];
			let padding = fieldPadding.get(key);
			if (typeof padding === 'undefined' || padding < value.length) {
				padding = value.length;
				fieldPadding.set(key, padding);
			}
			content += ` <span style="color:${color}">${key}</span>=${value}${'&nbsp;'.repeat(padding - value.length)}`;
		}
		content += '<br />';
	});
	return content;
};

// logAppender is a state updater function, which appends the new log chunks to the existing ones.
// In case the prev chunk array's last element doesn't have limit number of log record elements,
// it will be extended.
const logAppender = (limit: number) => (update: Array<Record>, prev: Array<Chunk>) => {
	const newChunks = [];
	let first = 0;
	let last = 0;
	let extended = 0;
	if (prev.length > 0 && prev[prev.length - 1].len < limit) {
		extended = 1;
		const l = Math.min(limit - prev[prev.length - 1].len, update.length);
		newChunks.push({
			content: prev[prev.length - 1].content + createLogChunk(update.slice(0, l)),
			t:       prev[prev.length - 1].t,
			len:     prev[prev.length - 1].len + l,
		});
		first = l;
		last = l;
	}
	while (last < update.length) {
		last = Math.min(update.length, last + limit);
		newChunks.push({
			content: createLogChunk(update.slice(first, last)),
			t:       update[first].t,
			len:     last - first,
		});
		first += limit;
	}
	return [...prev.slice(0, prev.length - extended), ...newChunks];
};

// defaultContent is the initial value of the state content.
const defaultContent: Content = {
	general: {
		version: null,
		commit:  null,
	},
	home:    {},
	chain:   {},
	txpool:  {},
	network: {},
	system:  {
		activeMemory:   [],
		virtualMemory:  [],
		networkIngress: [],
		networkEgress:  [],
		processCPU:     [],
		systemCPU:      [],
		diskRead:       [],
		diskWrite:      [],
	},
	logs:    {
		chunk: [],
	},
};

// updaters contains the state updater functions for each path of the state.
//
// TODO (kurkomisi): Define a tricky type which embraces the content and the updaters.
const updaters = {
	general: {
		version: replacer,
		commit:  replacer,
	},
	home:    null,
	chain:   null,
	txpool:  null,
	network: null,
	system:  {
		activeMemory:   appender(200),
		virtualMemory:  appender(200),
		networkIngress: appender(200),
		networkEgress:  appender(200),
		processCPU:     appender(200),
		systemCPU:      appender(200),
		diskRead:       appender(200),
		diskWrite:      appender(200),
	},
	logs: {
		chunk: logAppender(50),
	},
};

// styles contains the constant styles of the component.
const styles = {
	dashboard: {
		display:  'flex',
		flexFlow: 'column',
		width:    '100%',
		height:   '100%',
		zIndex:   1,
		overflow: 'hidden',
	},
	logChunk: {
		color:      'white',
		fontFamily: 'monospace',
	},
};

// themeStyles returns the styles generated from the theme for the component.
const themeStyles: Object = (theme: Object) => ({
	dashboard: {
		background: theme.palette.background.default,
	},
});

export type Props = {
	classes: Object, // injected by withStyles()
};

type State = {
	active: string, // active menu
	sideBar: boolean, // true if the sidebar is opened
	content: Content, // the visualized data
	shouldUpdate: Object, // labels for the components, which need to re-render based on the incoming message
	server: ?WebSocket,
};

// Dashboard is the main component, which renders the whole page, makes connection with the server and
// listens for messages. When there is an incoming message, updates the page's content correspondingly.
class Dashboard extends Component<Props, State> {
	constructor(props: Props) {
		super(props);
		this.state = {
			active:       MENU.get('home').id,
			sideBar:      true,
			content:      defaultContent,
			shouldUpdate: {},
			server:       null,
		};
	}

	// componentDidMount initiates the establishment of the first websocket connection after the component is rendered.
	componentDidMount() {
		this.reconnect();
	}

	// reconnect establishes a websocket connection with the server, listens for incoming messages
	// and tries to reconnect on connection loss.
	reconnect = () => {
		// PROD is defined by webpack.
		const server = new WebSocket(`${((window.location.protocol === 'https:') ? 'wss://' : 'ws://')}${PROD ? window.location.host : 'localhost:8080'}/api`);
		server.onopen = () => {
			this.setState({content: defaultContent, shouldUpdate: {}, server});
		};
		server.onmessage = (event) => {
			const msg: $Shape<Content> = JSON.parse(event.data);
			if (!msg) {
				console.error(`Incoming message is ${msg}`);
				return;
			}
			this.update(msg);
		};
		server.onclose = () => {
			this.setState({server: null});
			setTimeout(this.reconnect, 3000);
		};
	};

	// server can be accessed only through this function for safety reasons.
	send = (msg: string) => {
		if (this.state.server != null) {
			this.state.server.send(msg);
		}
	};

	// update updates the content corresponding to the incoming message.
	update = (msg: $Shape<Content>) => {
		this.setState(prevState => ({
			content:      deepUpdate(updaters, msg, prevState.content),
			shouldUpdate: shouldUpdate(updaters, msg),
		}));
	};

	// changeContent sets the active label, which is used at the content rendering.
	changeContent = (newActive: string) => {
		this.setState(prevState => (prevState.active !== newActive ? {active: newActive} : {}));
	};

	// switchSideBar opens or closes the sidebar's state.
	switchSideBar = () => {
		this.setState(prevState => ({sideBar: !prevState.sideBar}));
	};

	// logsHTML visualizes the log chunks. It is more efficient to insert pure HTML into the component, than
	// to create individual component for each log record and track them all. It also postpones the OOM issue.
	logsHTML = () => (
		<List>
			{this.state.content.logs.chunk.map((c, index) => (
				<ListItem key={index}>
					<div style={styles.logChunk} dangerouslySetInnerHTML={{__html: c.content}} />
				</ListItem>
			))}
		</List>
	);

	render() {
		return (
			<div className={this.props.classes.dashboard} style={styles.dashboard}>
				<Header
					switchSideBar={this.switchSideBar}
				/>
				<Body
					opened={this.state.sideBar}
					changeContent={this.changeContent}
					active={this.state.active}
					content={this.state.content}
					shouldUpdate={this.state.shouldUpdate}
					send={this.send}
					logs={this.logsHTML}
				/>
			</div>
		);
	}
}

export default withStyles(themeStyles)(Dashboard);
