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

import List, {ListItem} from 'material-ui/List';
import type {Record, Content, LogsMessage, Logs as LogsType} from '../types/content';

// if the scroll position is closer to the top/bottom than this value, the client sends a request for a new log chunk.
const requestLimit = 100;

// fieldPadding is a global map with maximum field value lengths seen until now
// to allow padding log contexts in a bit smarter way.
const fieldPadding = new Map();

// createChunk creates an HTML formatted object, which displays the given array similarly to
// the server side terminal.
const createChunk = (records: Array<Record>) => {
	let content = '';
	records.forEach((record) => {
		const {t, ctx} = record;
		let {lvl, msg} = record;
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

// inserter is a state updater function for the main component, which inserts the new log chunk into the chunk array.
// limit is the maximum length of the chunk array, used in order to prevent the OOM in the browser.
export const inserter = (limit: number) => (update: LogsMessage, prev: LogsType) => {
	prev.topChanged = 0;
	prev.bottomChanged = 0;
	if (!update.stream && update.end) {
		if (update.past) {
			prev.endTop = true;
		} else {
			prev.endBottom = true;
		}
		return prev;
	}
	if (update.stream && !prev.endBottom) {
		return prev;
	}
	if (!Array.isArray(update.chunk) || update.chunk.length < 1) {
		return prev;
	}
	const chunk = {
		content: createChunk(update.chunk),
		tFirst:  update.chunk[0].t,
		tLast:   update.chunk[update.chunk.length - 1].t,
	};
	if (!Array.isArray(prev.chunks) || prev.chunks.length < 1) {
		prev.chunks = [chunk];
		prev.topChanged = 1;
		prev.bottomChanged = 1;
		return prev;
	}
	if (update.stream) {
		// The stream chunks are appended to the last chunk, because otherwise the small stream chunks would cause
		// imbalance in the amount of the visualized logs. In order to protect a chunk from growing too large a new
		// chunk is created when a new file is opened on the server side. In case of stream end indicates if a new file
		// was opened.
		if (update.end) {
			if (prev.chunks.length >= limit) {
				prev.endTop = false;
				prev.chunks.splice(0, prev.chunks.length - limit + 1);
				prev.topChanged = -1;
			}
			prev.chunks = [...prev.chunks, chunk];
		} else {
			prev.chunks[prev.chunks.length - 1].content += chunk.content;
			prev.chunks[prev.chunks.length - 1].tLast = chunk.tLast;
		}
		prev.bottomChanged = 1;
		return prev;
	}
	if (update.past) {
		if (prev.chunks.length >= limit) {
			prev.endBottom = false;
			prev.chunks.splice(limit - 1, prev.chunks.length - limit + 1);
			prev.bottomChanged = -1;
		}
		prev.chunks = [chunk, ...prev.chunks];
		prev.topChanged = 1;
		return prev;
	}
	if (prev.chunks.length >= limit) {
		prev.endTop = false;
		prev.chunks.splice(0, prev.chunks.length - limit + 1);
		prev.topChanged = -1;
	}
	prev.chunks = [...prev.chunks, chunk];
	prev.bottomChanged = 1;
	return prev;
};

// styles contains the constant styles of the component.
const styles = {
	logs: {
		overflowX: 'auto',
	},
	logListItem: {
		padding: 0,
	},
	logChunk: {
		color:      'white',
		fontFamily: 'monospace',
	},
};

export type Props = {
	container:    Object,
	content:      Content,
	shouldUpdate: Object,
	send:         string => void,
};

type State = {
	requestAllowed: boolean,
};

// Logs renders the log page.
class Logs extends Component<Props, State> {
	constructor(props: Props) {
		super(props);
		this.state = {
			requestAllowed: true,
		};
	}

	// onScroll is triggered by the parent component's scroll event, and sends requests if the scroll position is
	// at the top or at the bottom.
	onScroll = () => {
		const {logs} = this.props.content;
		if (typeof this.props.container === 'undefined' || logs.chunks.length < 1 || !this.state.requestAllowed) {
			return;
		}
		if (this.atTop() && !logs.endTop) {
			this.props.send(JSON.stringify({
				Logs: {
					Time: logs.chunks[0].tFirst,
					Past: true,
				},
			}));
			this.setState({requestAllowed: false});
		}
		if (this.atBottom() && !logs.endBottom) {
			this.props.send(JSON.stringify({
				Logs: {
					Time: logs.chunks[logs.chunks.length - 1].tLast,
					Past: false,
				},
			}));
			this.setState({requestAllowed: false});
		}
	};

	// atTop checks if the scroll position it at the top of the container.
	atTop = () => this.props.container.scrollTop <= requestLimit;

	// atBottom checks if the scroll position it at the bottom of the container.
	atBottom = () =>
		this.props.container.scrollHeight - this.props.container.scrollTop <=
		this.props.container.clientHeight + requestLimit;

	// didUpdate is called by the parent component, which provides the container. Sends the first request if the
	// visible part of the container isn't full, and resets the scroll position in order to avoid jumping when new
	// chunk is inserted.
	didUpdate = () => {
		if (typeof this.props.shouldUpdate.logs === 'undefined' || typeof this.content === 'undefined') {
			return;
		}
		const {logs} = this.props.content;
		const {container} = this.props;
		if (typeof container === 'undefined' || logs.chunks.length < 1) {
			return;
		}
		this.setState({requestAllowed: true});
		if (this.content.clientHeight < container.clientHeight) {
			// Only enters here at the beginning, when there isn't enough log to fill the container.
			//
			// In case there isn't any log chunk in the array, a request with time (new Date()).toISOString()
			// could be sent, but it would allow to duplicate the first few records from the stream, since the
			// stream handler loads the last file. No log records will appear before the first stream chunk.
			if (!logs.endTop) {
				this.props.send(JSON.stringify({
					Logs: {
						Time: logs.chunks[0].tFirst,
						Past: true,
					},
				}));
				this.setState({requestAllowed: false});
			}
			return;
		}
		const chunks = this.content.children[0].children;
		if (this.atTop()) {
			if (logs.topChanged > 0) {
				container.scrollTop = chunks[0].clientHeight;
			}
			return;
		}
		if (this.atBottom() && logs.bottomChanged > 0) {
			if (logs.endBottom) {
				container.scrollTop = container.scrollHeight - container.clientHeight;
			} else {
				container.scrollTop = container.scrollHeight - chunks[chunks.length - 1].clientHeight;
			}
		}
	};

	render() {
		return (
			<div style={styles.logs} ref={(ref) => { this.content = ref; }}>
				<List>
					{this.props.content.logs.chunks.map((c, index) => (
						<ListItem style={styles.logListItem} key={index}>
							<div style={styles.logChunk} dangerouslySetInnerHTML={{__html: c.content}} />
						</ListItem>
					))}
				</List>
			</div>
		);
	}
}

export default Logs;
