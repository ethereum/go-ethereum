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

import {MENU, DURATION} from '../common';

// styles contains the constant styles of the component.
const styles = {
	menu: {
		default: {
			transition: `margin-left ${DURATION}ms`,
		},
		transition: {
			entered: {marginLeft: -200},
		},
	},
};

// themeStyles returns the styles generated from the theme for the component.
const themeStyles = theme => ({
	list: {
		background: theme.palette.grey[900],
	},
	listItem: {
		minWidth: theme.spacing.unit * 7,
	},
	icon: {
		fontSize: theme.spacing.unit * 3,
	},
});

export type Props = {
	classes: Object, // injected by withStyles()
	opened: boolean,
	changeContent: string => void,
};

// SideBar renders the sidebar of the dashboard.
class SideBar extends Component<Props> {
	shouldComponentUpdate(nextProps) {
		return nextProps.opened !== this.props.opened;
	}

	// clickOn returns a click event handler function for the given menu item.
	clickOn = menu => (event) => {
		event.preventDefault();
		this.props.changeContent(menu);
	};

	// menuItems returns the menu items corresponding to the sidebar state.
	menuItems = (transitionState) => {
		const {classes} = this.props;
		const children = [];
		MENU.forEach((menu) => {
			children.push((
				<ListItem button key={menu.id} onClick={this.clickOn(menu.id)} className={classes.listItem}>
					<ListItemIcon>
						<Icon className={classes.icon}>
							<FontAwesome name={menu.icon} />
						</Icon>
					</ListItemIcon>
					<ListItemText
						primary={menu.title}
						style={{
							...styles.menu.default,
							...styles.menu.transition[transitionState],
							padding: 0,
						}}
					/>
				</ListItem>
			));
		});
		return children;
	};

	// menu renders the list of the menu items.
	menu = (transitionState: Object) => (
		<div className={this.props.classes.list}>
			<List>
				{this.menuItems(transitionState)}
			</List>
		</div>
	);

	render() {
		return (
			<Transition mountOnEnter in={this.props.opened} timeout={{enter: DURATION}}>
				{this.menu}
			</Transition>
		);
	}
}

export default withStyles(themeStyles)(SideBar);
