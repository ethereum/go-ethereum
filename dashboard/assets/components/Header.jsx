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

import withStyles from '@material-ui/core/styles/withStyles';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import IconButton from '@material-ui/core/IconButton';
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {faBars, faSortAmountUp, faClock, faUsers, faSync} from '@fortawesome/free-solid-svg-icons';
import Typography from '@material-ui/core/Typography';
import type {Content} from '../types/content';


const magnitude = [31536000, 604800, 86400, 3600, 60, 1];
const label     = ['y', 'w', 'd', 'h', 'm', 's'];

// styles contains the constant styles of the component.
const styles = {
	header: {
		height: '8%',
	},
	headerText: {
		marginRight: 15,
	},
	toolbar: {
		height:    '100%',
		minHeight: 'unset',
	},
};

// themeStyles returns the styles generated from the theme for the component.
const themeStyles = (theme: Object) => ({
	header: {
		backgroundColor: theme.palette.grey[900],
		color:           theme.palette.getContrastText(theme.palette.grey[900]),
		zIndex:          theme.zIndex.appBar,
	},
	toolbar: {
		paddingLeft:  theme.spacing.unit,
		paddingRight: theme.spacing.unit,
	},
	title: {
		paddingLeft: theme.spacing.unit,
		fontSize:    3 * theme.spacing.unit,
		flex:        1,
	},
});

export type Props = {
	classes:       Object, // injected by withStyles()
	switchSideBar: () => void,
	content:       Content,
	networkID:     number,
};

type State = {
	since: string,
}
// Header renders the header of the dashboard.
class Header extends Component<Props, State> {
	constructor(props) {
		super(props);
		this.state = {since: ''};
	}

	componentDidMount() {
		this.interval = setInterval(() => this.setState(() => {
			// time (seconds) since last block.
			let timeDiff = Math.floor((Date.now() - this.props.content.chain.currentBlock.timestamp * 1000) / 1000);
			let since = '';
			let i = 0;
			for (; i < magnitude.length && timeDiff < magnitude[i]; i++);
			for (let j = 2; i < magnitude.length && j > 0; j--, i++) {
				const t = Math.floor(timeDiff / magnitude[i]);
				if (t > 0) {
					since += `${t}${label[i]} `;
					timeDiff %= magnitude[i];
				}
			}
			if (since === '') {
				since = 'now';
			}
			this.setState({since: since});
		}), 1000);
	}

	componentWillUnmount() {
		clearInterval(this.interval);
	}

	render() {
		const {classes} = this.props;

		return (
			<AppBar position='static' className={classes.header} style={styles.header}>
				<Toolbar className={classes.toolbar} style={styles.toolbar}>
					<IconButton onClick={this.props.switchSideBar}>
						<FontAwesomeIcon icon={faBars} />
					</IconButton>
					<Typography type='title' color='inherit' noWrap className={classes.title}>
						Go Ethereum Dashboard
					</Typography>
					<Typography style={styles.headerText}>
						<FontAwesomeIcon icon={faSortAmountUp} /> {this.props.content.chain.currentBlock.number}
					</Typography>
					<Typography style={styles.headerText}>
						<FontAwesomeIcon icon={faClock} /> {this.state.since}
					</Typography>
					<Typography style={styles.headerText}>
						<FontAwesomeIcon icon={faUsers} /> {this.props.content.network.activePeerCount}
					</Typography>
				</Toolbar>
			</AppBar>
		);
	}
}

export default withStyles(themeStyles)(Header);
