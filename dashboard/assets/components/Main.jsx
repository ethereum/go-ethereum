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
import classNames from 'classnames';
import {withStyles} from 'material-ui/styles';

import {TAGS, DRAWER_WIDTH} from "./Common.jsx";
import Home from './Home.jsx';

// ContentSwitch chooses and renders the proper page content.
class ContentSwitch extends Component {
    render() {
        switch(this.props.active) {
            case TAGS.home.id:
                return <Home memory={this.props.memory} traffic={this.props.traffic} shouldUpdate={this.props.shouldUpdate} />;
            case TAGS.chain.id:
                return null;
            case TAGS.transactions.id:
                return null;
            case TAGS.network.id:
                // Only for testing.
                return null;
            case TAGS.system.id:
                return null;
            case TAGS.logs.id:
                return <div>{this.props.logs.map((log, index) => <div key={index}>{log}</div>)}</div>;
        }
        return null;
    }
}

ContentSwitch.propTypes = {
    active:       PropTypes.string.isRequired,
    shouldUpdate: PropTypes.object.isRequired,
};

// styles contains the styles for the Main component.
const styles = theme => ({
    content: {
        width:           '100%',
        marginLeft:      -DRAWER_WIDTH,
        flexGrow:        1,
        backgroundColor: theme.palette.background.default,
        padding:         theme.spacing.unit * 3,
        transition:      theme.transitions.create('margin', {
            easing:   theme.transitions.easing.sharp,
            duration: theme.transitions.duration.leavingScreen,
        }),
        marginTop:                    56,
        overflow:                     'auto',
        [theme.breakpoints.up('sm')]: {
            content: {
                height:    'calc(100% - 64px)',
                marginTop: 64,
            },
        },
    },
    contentShift: {
        marginLeft: 0,
        transition: theme.transitions.create('margin', {
            easing:   theme.transitions.easing.easeOut,
            duration: theme.transitions.duration.enteringScreen,
        }),
    },
});

// Main renders a component for the page content.
class Main extends Component {
    render() {
        // The classes property is injected by withStyles().
        const {classes} = this.props;

        return (
            <main className={classNames(classes.content, this.props.opened && classes.contentShift)}>
                <ContentSwitch
                    active={this.props.active}
                    memory={this.props.memory}
                    traffic={this.props.traffic}
                    logs={this.props.logs}
                    shouldUpdate={this.props.shouldUpdate}
                />
            </main>
        );
    }
}

Main.propTypes = {
    classes:      PropTypes.object.isRequired,
    opened:       PropTypes.bool.isRequired,
    active:       PropTypes.string.isRequired,
    shouldUpdate: PropTypes.object.isRequired,
};

export default withStyles(styles)(Main);
