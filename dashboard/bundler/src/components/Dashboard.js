import Component from 'component';
import {isNullOrUndefined, MEMORY_SAMPLE_LIMIT} from "./Common";
import {SideBar} from "./SideBar";
import {TopNavigation} from "./TopNavigation";
import PageContent from './PageContent';
import {Footer} from './Footer';

// Dashboard is the main component, which renders the whole page,
// makes connection with the server and listens for messages.
// When there is an incoming message, updates the page's content correspondingly.
export default class Dashboard extends Component {
    constructor(props) {
        super(props);
        this.state = {
            charts: { // Stores the state of the charts.
                memory: {
                    labels: [],
                    datasets: [{
                        label: "system/memory/inuse",
                        backgroundColor: "rgba(38, 185, 154, 0.31)",
                        borderColor: "rgba(38, 185, 154, 0.7)",
                        pointBorderColor: "rgba(38, 185, 154, 0.7)",
                        pointBackgroundColor: "rgba(38, 185, 154, 0.7)",
                        pointHoverBackgroundColor: "#fff",
                        pointHoverBorderColor: "rgba(220,220,220,1)",
                        pointBorderWidth: 1,
                        data: [],
                    }],
                },
                traffic: {
                    labels: [],
                    datasets: [{
                        label: "p2p/InboundTraffic",
                        backgroundColor: "rgba(3, 88, 106, 0.3)",
                        borderColor: "rgba(3, 88, 106, 0.70)",
                        pointBorderColor: "rgba(3, 88, 106, 0.70)",
                        pointBackgroundColor: "rgba(3, 88, 106, 0.70)",
                        pointHoverBackgroundColor: "#fff",
                        pointHoverBorderColor: "rgba(151,187,205,1)",
                        pointBorderWidth: 1,
                        data: [],
                    }],
                },
            },
        };
    }

    // updateCharts Analyzes the incoming message, and updates the charts' content correspondingly.
    updateCharts = msg => {
        const memory = this.state.charts.memory;
        const traffic = this.state.charts.traffic;

        // Fill the dashboard with the past data. metrics is set only in the first msg,
        // after the connection is established.
        if (msg.metrics !== undefined) {
            // Clear the arrays to prevent data confusion with the previous connection.
            memory.labels = [];
            traffic.labels = [];
            memory.datasets[0].data = [];
            traffic.datasets[0].data = [];

            const mem = msg.metrics.memory;
            const traff = msg.metrics.processor; // TODO (kurkomisi): !!!

            // Put the past data to the beginning of the arrays. This prevents confusion with the next msg data,
            // which goes to the end.
            for (let i = mem.length - 1; i >= 0 && MEMORY_SAMPLE_LIMIT > memory.labels.length; --i) {
                memory.labels.unshift(mem[i].time.substring(mem[i].time.length - 5));
                traffic.labels.unshift(mem[i].time.substring(mem[i].time.length - 5));
                memory.datasets[0].data.unshift(mem[i].value);
                traffic.datasets[0].data.unshift(traff[i].value);
            }

            this.setState({charts: {memory, traffic,}}); // Update the components.
            return;
        }

        // Put the new data to the end of the arrays.
        if (msg.memory !== undefined) {
            // Remove the first elements in case the samples' amount exceeds the limit.
            if (memory.labels.length === MEMORY_SAMPLE_LIMIT) {
                memory.labels.shift();
                traffic.labels.shift();
                memory.datasets[0].data.shift();
                traffic.datasets[0].data.shift();
            }
            memory.labels.push(msg.memory.time.substring(msg.memory.time.length - 5));
            traffic.labels.push(msg.memory.time.substring(msg.memory.time.length - 5));
            memory.datasets[0].data.push(msg.memory.value);
            traffic.datasets[0].data.push(msg.processor.value);

            this.setState({charts: {memory, traffic,}}); // Update the components.
        }
    };

    // reconnect establishes a websocket connection with the server, listens for incoming messages
    // and tries to reconnect on connection loss.
    reconnect = () => {
        const server = new WebSocket("ws://" + location.host + "/api");

        server.onmessage = event => {
            const msg = JSON.parse(event.data);
            if (isNullOrUndefined(msg)) {
                return;
            }
            this.updateCharts(msg);
        };

        server.onclose = () => setTimeout(this.reconnect, 3000);
    };

    // componentDidMount initiates the establishment of the first websocket connection after the component is rendered.
    componentDidMount = () => this.reconnect();

    // render renders the components of the dashboard.
    render = () => <div className="container body">
        <div className="main_container">
            <SideBar/>
            <TopNavigation/>
            <PageContent charts={this.state.charts}/>
            <Footer/>
        </div>
    </div>;
}