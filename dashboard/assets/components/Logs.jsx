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

// requestBand says how wide is the top/bottom zone, eg. 0.1 means 10% of the container height.
const requestBand = 0.05;

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
// limit is the maximum length of the chunk array, used in order to prevent the browser from OOM.
export const inserter = (limit: number) => (update: LogsMessage, prev: LogsType) => {
	prev.topChanged = 0;
	prev.bottomChanged = 0;
	if (!Array.isArray(update.chunk) || update.chunk.length < 1) {
		return prev;
	}
	if (!Array.isArray(prev.chunks)) {
		prev.chunks = [];
	}
	const content = createChunk(update.chunk);
	if (!update.old) {
		// In case of stream chunk.
		if (!prev.endBottom) {
			return prev;
		}
		if (prev.chunks.length < 1) {
			// This should never happen, because the first chunk is always a non-stream chunk.
			return [{content, name: '00000000000000.log'}];
		}
		prev.chunks[prev.chunks.length - 1].content += content;
		prev.bottomChanged = 1;
		return prev;
	}
	const chunk = {
		content,
		name: update.old.name,
	};
	if (update.old.past) {
		if (update.old.last) {
			prev.endTop = true;
		}
		if (prev.chunks.length >= limit) {
			prev.endBottom = false;
			prev.chunks.splice(limit - 1, prev.chunks.length - limit + 1);
			prev.bottomChanged = -1;
		}
		prev.chunks = [chunk, ...prev.chunks];
		prev.topChanged = 1;
		return prev;
	}
	if (update.old.last) {
		prev.endBottom = true;
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
		this.content = React.createRef();
		this.state = {
			requestAllowed: true,
		};
	}

	componentDidMount() {
		const {container} = this.props;
		container.scrollTop = container.scrollHeight - container.clientHeight;
	}

	// onScroll is triggered by the parent component's scroll event, and sends requests if the scroll position is
	// at the top or at the bottom.
	onScroll = () => {
		if (!this.state.requestAllowed || typeof this.content === 'undefined') {
			return;
		}
		const {logs} = this.props.content;
		if (logs.chunks.length < 1) {
			return;
		}
		if (this.atTop()) {
			if (!logs.endTop) {
				this.setState({requestAllowed: false});
				this.props.send(JSON.stringify({
					Logs: {
						Name: logs.chunks[0].name,
						Past: true,
					},
				}));
			}
		} else if (this.atBottom()) {
			if (!logs.endBottom) {
				this.setState({requestAllowed: false});
				this.props.send(JSON.stringify({
					Logs: {
						Name: logs.chunks[logs.chunks.length - 1].name,
						Past: false,
					},
				}));
			}
		}
	};

	// atTop checks if the scroll position it at the top of the container.
	atTop = () => this.props.container.scrollTop <= this.props.container.scrollHeight * requestBand;

	// atBottom checks if the scroll position it at the bottom of the container.
	atBottom = () => {
		const {container} = this.props;
		return container.scrollHeight - container.scrollTop <=
			container.clientHeight + container.scrollHeight * requestBand;
	};

	// beforeUpdate is called by the parent component, saves the previous scroll position
	// and the height of the first log chunk, which can be deleted during the insertion.
	beforeUpdate = () => {
		let firstHeight = 0;
		if (this.content && this.content.children[0] && this.content.children[0].children[0]) {
			firstHeight = this.content.children[0].children[0].clientHeight;
		}
		return {
			scrollTop: this.props.container.scrollTop,
			firstHeight,
		};
	};

	// didUpdate is called by the parent component, which provides the container. Sends the first request if the
	// visible part of the container isn't full, and resets the scroll position in order to avoid jumping when new
	// chunk is inserted.
	didUpdate = (prevProps, prevState, snapshot) => {
		if (typeof this.props.shouldUpdate.logs === 'undefined' || typeof this.content === 'undefined' || snapshot === null) {
			return;
		}
		const {logs} = this.props.content;
		const {container} = this.props;
		if (typeof container === 'undefined' || logs.chunks.length < 1) {
			return;
		}
		if (this.content.clientHeight < container.clientHeight) {
			// Only enters here at the beginning, when there isn't enough log to fill the container
			// and the scroll bar doesn't appear.
			if (!logs.endTop) {
				this.setState({requestAllowed: false});
				this.props.send(JSON.stringify({
					Logs: {
						Name: logs.chunks[0].name,
						Past: true,
					},
				}));
			}
			return;
		}
		const chunks = this.content.children[0].children;
		let {scrollTop} = snapshot;
		if (logs.topChanged > 0) {
			scrollTop += chunks[0].clientHeight;
		} else if (logs.bottomChanged > 0) {
			if (logs.topChanged < 0) {
				scrollTop -= snapshot.firstHeight;
			} else if (logs.endBottom && this.atBottom()) {
				scrollTop = container.scrollHeight - container.clientHeight;
			}
		}
		container.scrollTop = scrollTop;
		this.setState({requestAllowed: true});
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
