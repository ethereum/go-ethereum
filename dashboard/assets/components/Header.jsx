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

import withStyles from 'material-ui/styles/withStyles';
import AppBar from 'material-ui/AppBar';
import Toolbar from 'material-ui/Toolbar';
import Transition from 'react-transition-group/Transition';
import IconButton from 'material-ui/IconButton';
import Typography from 'material-ui/Typography';
import ChevronLeftIcon from 'material-ui-icons/ChevronLeft';

import {DURATION} from '../common';

// styles contains the constant styles of the component.
const styles = {
	arrow: {
		default: {
			transition: `transform ${DURATION}ms`,
		},
		transition: {
			entered: {transform: 'rotate(180deg)'},
		},
	},
};

// themeStyles returns the styles generated from the theme for the component.
const themeStyles = (theme: Object) => ({
	header: {
		backgroundColor: theme.palette.background.appBar,
		color:           theme.palette.getContrastText(theme.palette.background.appBar),
		zIndex:          theme.zIndex.appBar,
	},
	toolbar: {
		paddingLeft:  theme.spacing.unit,
		paddingRight: theme.spacing.unit,
	},
	title: {
		paddingLeft: theme.spacing.unit,
	},
});

export type Props = {
	classes: Object, // injected by withStyles()
	opened: boolean,
	switchSideBar: () => void,
};

// Header renders the header of the dashboard.
class Header extends Component<Props> {
	shouldComponentUpdate(nextProps) {
		return nextProps.opened !== this.props.opened;
	}

	// arrow renders a button, which changes the sidebar's state.
	arrow = (transitionState: string) => (
		<IconButton onClick={this.props.switchSideBar}>
			<ChevronLeftIcon
				style={{
					...styles.arrow.default,
					...styles.arrow.transition[transitionState],
				}}
			/>
		</IconButton>
	);

	render() {
		const {classes, opened} = this.props;

		return (
			<AppBar position='static' className={classes.header}>
				<Toolbar className={classes.toolbar}>
					<Transition mountOnEnter in={opened} timeout={{enter: DURATION}}>
						{this.arrow}
					</Transition>
					<Typography type='title' color='inherit' noWrap className={classes.title}>
						Go Ethereum Dashboard
					</Typography>
				</Toolbar>
			</AppBar>
		);
	}
}

export default withStyles(themeStyles)(Header);
