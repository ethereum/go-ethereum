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
import PropTypes from 'prop-types';

import withStyles from 'material-ui/styles/withStyles';
import List, {ListItem, ListItemIcon, ListItemText} from 'material-ui/List';
import Icon from 'material-ui/Icon';
import Transition from 'react-transition-group/Transition';
import {Icon as FontAwesome} from 'react-fa'

import {TAGS, DURATION} from './Common.jsx';

// menuDefault is the default style of the menu.
const menuDefault = {
    transition: `margin-left ${DURATION}ms`,
};
// menu Transition is the additional style of the menu corresponding to the transition's state.
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

// SideBar renders the sidebar of the dashboard.
@withStyles(styles)
class SideBar extends Component {
    constructor(props) {
        super(props);

        // clickOn contains onClick event functions for the menu items.
        // Instantiate only once, and reuse the existing functions to prevent the creation of
        // new function instances every time the render method is triggered.
        this.clickOn = {};
        for(let key in TAGS) {
            const id = TAGS[key].id;
            this.clickOn[id] = event => {
                event.preventDefault();
                props.changeContent(id);
            };
        }
    }

    shouldComponentUpdate(nextProps) {
        return nextProps.opened !== this.props.opened;
    }

    // menu renders the list of the menu items.
    menu = transitionState => {
        const {classes} = this.props; // The classes property is injected by withStyles().

        return (
            <div className={classes.list}>
                <List>
                    {
                        Object.values(TAGS).map(tag => (
                            <ListItem button key={tag.id} onClick={this.clickOn[tag.id]} className={classes.listItem}>
                                <ListItemIcon>
                                    <Icon className={classes.icon}>
                                        <FontAwesome name={tag.icon} />
                                    </Icon>
                                </ListItemIcon>
                                <ListItemText
                                    primary={tag.title}
                                    style={{
                                        ...menuDefault,
                                        ...menuTransition[transitionState],
                                        padding: 0,
                                    }}
                                />
                            </ListItem>
                        ))
                    }
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

SideBar.propTypes = {
    opened:        PropTypes.bool.isRequired,
    changeContent: PropTypes.func.isRequired,
};

export default SideBar;
