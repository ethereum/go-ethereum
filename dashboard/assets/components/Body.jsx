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

import SideBar from './SideBar.jsx';
import Content from "./Content.jsx";

// Styles for the Body component.
const styles = theme => ({
    body: {
        display: 'flex',
        width:   '100%',
        height:  '100%',
    },
});

// Body renders the body of the dashboard.
@withStyles(styles)
class Body extends Component {
    render() {
        const {classes} = this.props; // The classes property is injected by withStyles().

        return (
            <div className={classes.body}>
                <SideBar
                    opened={this.props.opened}
                    changeContent={this.props.changeContent}
                />
                <Content
                    active={this.props.active}
                    memory={this.props.memory}
                    traffic={this.props.traffic}
                    logs={this.props.logs}
                    shouldUpdate={this.props.shouldUpdate}
                />
            </div>
        );
    }
}

Body.propTypes = {
    opened:        PropTypes.bool.isRequired,
    changeContent: PropTypes.func.isRequired,
    active:        PropTypes.string.isRequired,
    memory:        PropTypes.array.isRequired,
    traffic:       PropTypes.array.isRequired,
    logs:          PropTypes.array.isRequired,
    shouldUpdate:  PropTypes.object.isRequired,
};

export default Body;
