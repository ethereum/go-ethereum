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

import Header from './Header.jsx';
import Body from './Body.jsx';
import {isNullOrUndefined, LIMIT, TAGS, DATA_KEYS} from "./Common.jsx";

// Styles for the Dashboard component.
const styles = theme => ({
    dashboard: {
        display:    'flex',
        flexFlow:   'column',
        width:      '100%',
        height:     '100%',
        background: theme.palette.background.default,
        zIndex:     1,
        overflow:   'hidden',
    },
});

// Dashboard is the main component, which renders the whole page, makes connection with the server and listens for messages.
// When there is an incoming message, updates the page's content correspondingly.
@withStyles(styles)
class Dashboard extends Component {
    constructor(props) {
        super(props);
        this.state = {
            active:       TAGS.home.id, // active menu
            sideBar:      true, // true if the sidebar is opened
            memory:       [],
            traffic:      [],
            logs:         [],
            shouldUpdate: {}, // contains the labels of the incoming sample types
        };
    }

    // componentDidMount initiates the establishment of the first websocket connection after the component is rendered.
    componentDidMount() {
        this.reconnect();
    }

    // reconnect establishes a websocket connection with the server, listens for incoming messages
    // and tries to reconnect on connection loss.
    reconnect = () => {
        const server = new WebSocket(((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host + "/api");

        server.onmessage = event => {
            const msg = JSON.parse(event.data);
            if (isNullOrUndefined(msg)) {
                return;
            }
            this.update(msg);
        };

        server.onclose = () => {
            setTimeout(this.reconnect, 3000);
        };
    };

    // update analyzes the incoming message, and updates the charts' content correspondingly.
    update = msg => {
        this.setState(prevState => {
            let newState = [];
            newState.shouldUpdate = {};
            const insert = (key, values, limit) => {
                newState[key] = [...prevState[key], ...values];
                while (newState[key].length > limit) {
                    newState[key].shift();
                }
                newState.shouldUpdate[key] = true;
            };
            // (Re)initialize the state with the past data.
            if (!isNullOrUndefined(msg.history)) {
                const memory = DATA_KEYS.memory;
                const traffic = DATA_KEYS.traffic;
                newState[memory] = [];
                newState[traffic] = [];
                if (!isNullOrUndefined(msg.history.memorySamples)) {
                    newState[memory] = msg.history.memorySamples.map(elem => isNullOrUndefined(elem.value) ? 0 : elem.value);
                    while (newState[memory].length > LIMIT.memory) {
                        newState[memory].shift();
                    }
                    newState.shouldUpdate[memory] = true;
                }
                if (!isNullOrUndefined(msg.history.trafficSamples)) {
                    newState[traffic] = msg.history.trafficSamples.map(elem => isNullOrUndefined(elem.value) ? 0 : elem.value);
                    while (newState[traffic].length > LIMIT.traffic) {
                        newState[traffic].shift();
                    }
                    newState.shouldUpdate[traffic] = true;
                }
            }
            // Insert the new data samples.
            if (!isNullOrUndefined(msg.memory)) {
                insert(DATA_KEYS.memory, [isNullOrUndefined(msg.memory.value) ? 0 : msg.memory.value], LIMIT.memory);
            }
            if (!isNullOrUndefined(msg.traffic)) {
                insert(DATA_KEYS.traffic, [isNullOrUndefined(msg.traffic.value) ? 0 : msg.traffic.value], LIMIT.traffic);
            }
            if (!isNullOrUndefined(msg.log)) {
                insert(DATA_KEYS.logs, [msg.log], LIMIT.log);
            }

            return newState;
        });
    };

    // changeContent sets the active label, which is used at the content rendering.
    changeContent = newActive => {
        this.setState(prevState => prevState.active !== newActive ? {active: newActive} : {});
    };

    openSideBar = () => {
        this.setState({sideBar: true});
    };

    closeSideBar = () => {
        this.setState({sideBar: false});
    };

    render() {
        const {classes} = this.props; // The classes property is injected by withStyles().

        return (
            <div className={classes.dashboard}>
                <Header
                    opened={this.state.sideBar}
                    openSideBar={this.openSideBar}
                    closeSideBar={this.closeSideBar}
                />
                <Body
                    opened={this.state.sideBar}
                    changeContent={this.changeContent}
                    active={this.state.active}
                    memory={this.state.memory}
                    traffic={this.state.traffic}
                    logs={this.state.logs}
                    shouldUpdate={this.state.shouldUpdate}
                />
            </div>
        );
    }
}

export default Dashboard;
