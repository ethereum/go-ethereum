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
import List, {ListItem, ListItemIcon, ListItemText} from 'material-ui/List';
import Icon from 'material-ui/Icon';
import Transition from 'react-transition-group/Transition';
import {Icon as FontAwesome} from 'react-fa';

import {MENU, DURATION} from './Common';

// menuDefault is the default style of the menu.
const menuDefault = {
	transition: `margin-left ${DURATION}ms`,
};
// menuTransition is the additional style of the menu corresponding to the transition's state.
const menuTransition = {
	entered: {marginLeft: -200},
};
// Styles for the SideBar component.
const styles = theme => ({
	list: {
		background: theme.palette.background.appBar,
	},
	listItem: {
		minWidth: theme.spacing.unit * 3,
	},
	icon: {
		fontSize: theme.spacing.unit * 3,
	},
});
export type Props = {
	classes: Object,
	opened: boolean,
	changeContent: () => {},
};
// SideBar renders the sidebar of the dashboard.
class SideBar extends Component<Props> {
	constructor(props) {
		super(props);

		// clickOn contains onClick event functions for the menu items.
		// Instantiate only once, and reuse the existing functions to prevent the creation of
		// new function instances every time the render method is triggered.
		this.clickOn = {};
		MENU.forEach((menu) => {
			this.clickOn[menu.id] = (event) => {
				event.preventDefault();
				props.changeContent(menu.id);
			};
		});
	}

	shouldComponentUpdate(nextProps) {
		return nextProps.opened !== this.props.opened;
	}

	menuItems = (transitionState) => {
		const {classes} = this.props;
		const children = [];
		MENU.forEach((menu) => {
			children.push(
				<ListItem button key={menu.id} onClick={this.clickOn[menu.id]} className={classes.listItem}>
					<ListItemIcon>
						<Icon className={classes.icon}>
							<FontAwesome name={menu.icon} />
						</Icon>
					</ListItemIcon>
					<ListItemText
						primary={menu.title}
						style={{
							...menuDefault,
							...menuTransition[transitionState],
							padding: 0,
						}}
					/>
				</ListItem>,
			);
		});
		return children;
	};

	// menu renders the list of the menu items.
	menu = (transitionState) => {
		const {classes} = this.props; // The classes property is injected by withStyles().

		return (
			<div className={classes.list}>
				<List>
					{this.menuItems(transitionState)}
				</List>
			</div>
		);
	};

	render() {
		return (
			<Transition mountOnEnter in={this.props.opened} timeout={{enter: DURATION}}>
				{this.menu}
			</Transition>
		);
	}
}

export default withStyles(styles)(SideBar);
