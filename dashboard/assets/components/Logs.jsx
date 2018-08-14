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

import List from '@material-ui/core/List';
import ListItem from '@material-ui/core/ListItem';
import escapeHtml from 'escape-html';
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
		const time = new Date(t);
		if (lvl === '' || !(time instanceof Date) || isNaN(time) || typeof msg !== 'string' || !Array.isArray(ctx)) {
			content += '<span style="color:#ce3c23">Invalid log record</span><br />';
			return;
		}
		if (ctx.length > 0) {
			msg += '&nbsp;'.repeat(Math.max(40 - msg.length, 0));
		}
		const month = `0${time.getMonth() + 1}`.slice(-2);
		const date = `0${time.getDate()}`.slice(-2);
		const hours = `0${time.getHours()}`.slice(-2);
		const minutes = `0${time.getMinutes()}`.slice(-2);
		const seconds = `0${time.getSeconds()}`.slice(-2);
		content += `<span style="color:${color}">${lvl}</span>[${month}-${date}|${hours}:${minutes}:${seconds}] ${msg}`;

		for (let i = 0; i < ctx.length; i += 2) {
			const key = escapeHtml(ctx[i]);
			const val = escapeHtml(ctx[i + 1]);
			let padding = fieldPadding.get(key);
			if (typeof padding !== 'number' || padding < val.length) {
				padding = val.length;
				fieldPadding.set(key, padding);
			}
			let p = '';
			if (i < ctx.length - 2) {
				p = '&nbsp;'.repeat(padding - val.length);
			}
			content += ` <span style="color:${color}">${key}</span>=${val}${p}`;
		}
		content += '<br />';
	});
	return content;
};

// ADDED, SAME and REMOVED are used to track the change of the log chunk array.
// The scroll position is set using these values.
export const ADDED = 1;
export const SAME = 0;
export const REMOVED = -1;

// inserter is a state updater function for the main component, which inserts the new log chunk into the chunk array.
// limit is the maximum length of the chunk array, used in order to prevent the browser from OOM.
export const inserter = (limit: number) => (update: LogsMessage, prev: LogsType) => {
	prev.topChanged = SAME;
	prev.bottomChanged = SAME;
	if (!Array.isArray(update.chunk) || update.chunk.length < 1) {
		return prev;
	}
	if (!Array.isArray(prev.chunks)) {
		prev.chunks = [];
	}
	const content = createChunk(update.chunk);
	if (!update.source) {
		// In case of stream chunk.
		if (!prev.endBottom) {
			return prev;
		}
		if (prev.chunks.length < 1) {
			// This should never happen, because the first chunk is always a non-stream chunk.
			return [{content, name: '00000000000000.log'}];
		}
		prev.chunks[prev.chunks.length - 1].content += content;
		prev.bottomChanged = ADDED;
		return prev;
	}
	const chunk = {
		content,
		name: update.source.name,
	};
	if (prev.chunks.length > 0 && update.source.name < prev.chunks[0].name) {
		if (update.source.last) {
			prev.endTop = true;
		}
		if (prev.chunks.length >= limit) {
			prev.endBottom = false;
			prev.chunks.splice(limit - 1, prev.chunks.length - limit + 1);
			prev.bottomChanged = REMOVED;
		}
		prev.chunks = [chunk, ...prev.chunks];
		prev.topChanged = ADDED;
		return prev;
	}
	if (update.source.last) {
		prev.endBottom = true;
	}
	if (prev.chunks.length >= limit) {
		prev.endTop = false;
		prev.chunks.splice(0, prev.chunks.length - limit + 1);
		prev.topChanged = REMOVED;
	}
	prev.chunks = [...prev.chunks, chunk];
	prev.bottomChanged = ADDED;
	return prev;
};

// styles contains the constant styles of the component.
const styles = {
	logListItem: {
		padding:    0,
		lineHeight: 1.231,
	},
	logChunk: {
		color:      'white',
		fontFamily: 'monospace',
		whiteSpace: 'nowrap',
		width:      0,
	},
	waitMsg: {
		textAlign:  'center',
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
		if (typeof container === 'undefined') {
			return;
		}
		container.scrollTop = container.scrollHeight - container.clientHeight;
		const {logs} = this.props.content;
		if (typeof this.content === 'undefined' || logs.chunks.length < 1) {
			return;
		}
		if (this.content.clientHeight < container.clientHeight && !logs.endTop) {
			this.sendRequest(logs.chunks[0].name, true);
		}
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
		if (this.atTop() && !logs.endTop) {
			this.sendRequest(logs.chunks[0].name, true);
		} else if (this.atBottom() && !logs.endBottom) {
			this.sendRequest(logs.chunks[logs.chunks.length - 1].name, false);
		}
	};

	sendRequest = (name: string, past: boolean) => {
		this.setState({requestAllowed: false});
		this.props.send(JSON.stringify({
			Logs: {
				Name: name,
				Past: past,
			},
		}));
	};

	// atTop checks if the scroll position it at the top of the container.
	atTop = () => this.props.container.scrollTop <= this.props.container.scrollHeight * requestBand;

	// atBottom checks if the scroll position it at the bottom of the container.
	atBottom = () => {
		const {container} = this.props;
		return container.scrollHeight - container.scrollTop
			<= container.clientHeight + container.scrollHeight * requestBand;
	};

	// beforeUpdate is called by the parent component, saves the previous scroll position
	// and the height of the first log chunk, which can be deleted during the insertion.
	beforeUpdate = () => {
		let firstHeight = 0;
		const chunkList = this.content.children[1];
		if (chunkList && chunkList.children[0]) {
			firstHeight = chunkList.children[0].clientHeight;
		}
		return {
			scrollTop: this.props.container.scrollTop,
			firstHeight,
		};
	};

	// didUpdate is called by the parent component, which provides the container. Sends the first request if the
	// visible part of the container isn't full, and resets the scroll position in order to avoid jumping when a
	// chunk is inserted or removed.
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
			// Only enters here at the beginning, when there aren't enough logs to fill the container
			// and the scroll bar doesn't appear.
			if (!logs.endTop) {
				this.sendRequest(logs.chunks[0].name, true);
			}
			return;
		}
		let {scrollTop} = snapshot;
		if (logs.topChanged === ADDED) {
			// It would be safer to use a ref to the list, but ref doesn't work well with HOCs.
			scrollTop += this.content.children[1].children[0].clientHeight;
		} else if (logs.bottomChanged === ADDED) {
			if (logs.topChanged === REMOVED) {
				scrollTop -= snapshot.firstHeight;
			} else if (this.atBottom() && logs.endBottom) {
				scrollTop = container.scrollHeight - container.clientHeight;
			}
		}
		container.scrollTop = scrollTop;
		this.setState({requestAllowed: true});
	};

	render() {
		return (
			<div ref={(ref) => { this.content = ref; }}>
				<div style={styles.waitMsg}>
					{this.props.content.logs.endTop ? 'No more logs.' : 'Waiting for server...'}
				</div>
				<List>
					{this.props.content.logs.chunks.map((c, index) => (
						<ListItem style={styles.logListItem} key={index}>
							<div style={styles.logChunk} dangerouslySetInnerHTML={{__html: c.content}} />
						</ListItem>
					))}
				</List>
				{this.props.content.logs.endBottom || <div style={styles.waitMsg}>Waiting for server...</div>}
			</div>
		);
	}
}

export default Logs;
