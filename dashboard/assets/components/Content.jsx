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

import Home from './Home.jsx';
import {TAGS} from './Common.jsx';

// Styles for the Content component.
const styles = theme => ({
    content: {
        flexGrow:        1,
        backgroundColor: theme.palette.background.default,
        padding:         theme.spacing.unit * 3,
        overflow:        'auto',
    },
});

// Content renders the chosen content.
@withStyles(styles)
class Content extends Component {
    render() {
        const {classes, active, memory, traffic, logs, shouldUpdate} = this.props;

        let content = null;
        switch(active) {
            case TAGS.home.id:
                content = <Home memory={memory} traffic={traffic} shouldUpdate={shouldUpdate} />;
                break;
            case TAGS.chain.id:
                content = <div>Chain is under construction.</div>;
                break;
            case TAGS.transactions.id:
                content = <div>Transactions is under construction.</div>;
                break;
            case TAGS.network.id:
                content = <div>Network is under construction.</div>;
                break;
            case TAGS.system.id:
                content = <div>System is under construction.</div>;
                break;
            case TAGS.logs.id:
                content = <div>{logs.map((log, index) => <div key={index}>{log}</div>)}</div>;
        }

        return <div className={classes.content}>{content}</div>;
    }
}

Content.propTypes = {
    active:       PropTypes.string.isRequired,
    shouldUpdate: PropTypes.object.isRequired,
};

export default Content;
