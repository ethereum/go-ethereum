import React, {Component} from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';

import SideBar from './SideBar.jsx';
import Header from './Header.jsx';
import Main from "./Main.jsx";
import {isNullOrUndefined, defaultZero, MEMORY_SAMPLE_LIMIT, TAGS} from "./Common.jsx";

// Styles for the Dashboard component.
const styles = theme => ({
    appFrame: {
        position: 'relative',
        display: 'flex',
        width: '100%',
        height: '100%',
        background: '#303030',
    },
});

// Dashboard is the main component, which renders the whole page,
// makes connection with the server and listens for messages.
// When there is an incoming message, updates the page's content correspondingly.
class Dashboard extends Component {
    constructor(props) {
        super(props);
        this.state = {
            active: TAGS.home.id, // active menu
            sideBar: true, // true if the sidebar is opened
            memory: [],
            traffic: [],
            logs: [],
        };
    }

    // componentDidMount initiates the establishment of the first websocket connection after the component is rendered.
    componentDidMount() {
        this.reconnect();
    }

    // reconnect establishes a websocket connection with the server, listens for incoming messages
    // and tries to reconnect on connection loss.
    reconnect = () => {
        const server = new WebSocket("ws://" + location.host + "/api");

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
        // (Re)initialize the state with the past data. metrics is set only in the first msg,
        // after the connection is established.
        if (!isNullOrUndefined(msg.metrics)) {
            let memory = [];
            let traffic = [];
            if (!isNullOrUndefined(msg.metrics.memory)) {
                memory = msg.metrics.memory.map(elem => ({memory: elem.value}));
                traffic = msg.metrics.processor.map(elem => ({traffic: defaultZero(elem.value)})) // TODO (kurkomisi): traffic != processor!!!
            }
            this.setState({
                memory: memory,
                traffic: traffic,
                logs: [],
            });
        }

        // Insert the new data samples.
        isNullOrUndefined(msg.memory) || this.setState(prevState => {
            let memory = prevState.memory;
            let traffic = prevState.traffic;
            // Remove the first elements in case the samples' amount exceeds the limit.
            if (memory.length === MEMORY_SAMPLE_LIMIT) {
                memory.shift();
                traffic.shift();
            }
            return ({
                memory: [...memory, {memory: msg.memory.value}],
                traffic: [...traffic, {traffic: defaultZero(msg.processor.value)}],
            });
        });

        // Insert the new log.
        isNullOrUndefined(msg.log) || this.setState(prevState => {
            let logs = prevState.logs;
            if(logs.length > 20) {
                logs.shift();
            }
            return {logs: [...logs, msg.log]};
        });
    };

    // The change of the active label on the SideBar component will trigger a new render in the Main component.
    changeContent = active => {
        this.state.active === active || this.setState({active: active});
    };

    openSideBar = () => {
        this.setState({sideBar: true});
    };

    closeSideBar = () => {
        this.setState({sideBar: false});
    };

    render() {
        // The classes property is injected by withStyles().
        const {classes} = this.props;

        return (
            <div className={classes.appFrame}>
                <Header
                    opened={this.state.sideBar}
                    open={this.openSideBar}
                />
                <SideBar
                    opened={this.state.sideBar}
                    close={this.closeSideBar}
                    changeContent={this.changeContent}
                />
                <Main
                    opened={this.state.sideBar}
                    active={this.state.active}
                    memory={this.state.memory}
                    traffic={this.state.traffic}
                    logs={this.state.logs}
                />
            </div>
        );
    }
}

Dashboard.propTypes = {
    classes: PropTypes.object.isRequired,
};

export default withStyles(styles)(Dashboard);