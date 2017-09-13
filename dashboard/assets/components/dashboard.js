Chart.defaults.global.legend = { enabled: false };

const Component = React.Component;
// const Component = Inferno.Component;

// isNullOrUndefined returns true if a variable is null or undefined.
let isNullOrUndefined = variable => variable === null || typeof variable === 'undefined';

// MenuItem renders an item for a Menu component and the belonging submenu items, if there is any.
class MenuItem extends Component {
    render = () => <li>
        <a>
            <i className={`fa ${this.props.className}`}/>
            {this.props.text}
            <div className="fa fa-chevron-down"/>
        </a>
        {
            // Render dropdown menu only if there are children.
            isNullOrUndefined(this.props.children) || <ul className="nav child_menu">
                {React.Children.map(this.props.children, child => <li>{child}</li>)}
            </ul>
        }
    </li>;
}

// Menu renders a menu component.
let Menu = () => <ul className="nav side-menu">
    <MenuItem className="fa-home" text="Home">
        <a href="dashboard1.html">Dashboard1</a>
        <a href="dashboard2.html">Dashboard2</a>
    </MenuItem>
    <MenuItem className="fa-edit" text="Networking">
        <a href="networking.html">Networking</a>
    </MenuItem>
    <MenuItem className="fa-desktop" text="Txpool">
        <a href="txpool.html">Txpool</a>
    </MenuItem>
    <MenuItem className="fa-table" text="Logs">
        <a href="logs.html">Logs</a>
    </MenuItem>
    <MenuItem className="fa-clone" text="Blockchain">
        <a href="blockchain1.html">Blockchain1</a>
        <a href="blockchain2.html">Blockchain2</a>
        <a href="blockchain3.html">Blockchain3</a>
        <a href="blockchain4.html">Blockchain4</a>
        <a href="blockchain5.html">Blockchain5</a>
    </MenuItem>
    <MenuItem className="fa-bar-chart-o" text="System"/>
</ul>;

let Clearfix = () => <div className="clearfix"/>;

// SideBar renders a sidebar component.
let SideBar = () => <div className="col-md-3 left_col">
    <div className="left_col scroll-view">
        <div className="navbar nav_title" style={{border: 0}}>
            <a href="dashboard.html" className="site_title"><i className="fa fa-paw"/> <span>Go Ethereum Dashboard</span></a>
        </div>
        <Clearfix/>
        <div id="sidebar-menu" className="main_menu_side hidden-print main_menu">
            <div className="menu_section">
                <Menu/>
            </div>
        </div>
    </div>
</div>;

// TopNavigation renders a top navigation component.
let TopNavigation = () => <div className="top_nav">
    <div className="nav_menu">
        <nav className="" role="navigation">
            <div className="nav toggle">
                <a id="_______menu_toggle"> {/* TODO (kurkomisi): Resize the main container on toggle */}
                    <i className="fa fa-bars"/>
                </a>
            </div>
        </nav>
    </div>
</div>;

// Chart name is already in use.
// ChartComponent renders a chart component and updates it, when the related data changes.
class ChartComponent extends Component {
    constructor(props) {
        super(props);
        this.state = {};
    }

    componentDidMount = () => this.state.chart = new Chart(this.data, {
        type: this.props.type,
        data: this.props.data,
    });

    render = () => {
        if (!isNullOrUndefined(this.state.chart)) {
            this.state.chart.update();
        }

        return (
            <div className={this.props.className}>
                <div className="x_panel">
                    <div className="x_title">
                        <h2>{this.props.text}</h2>
                        <ul className="nav navbar-right panel_toolbox">
                            <li><a className="collapse-link"><i className="fa fa-chevron-up"/></a></li>
                            {
                                // Render dropdown menu only if there are children.
                                isNullOrUndefined(this.props.children) || <li className="dropdown">
                                    <a href="#" className="dropdown-toggle" data-toggle="dropdown" role="button"
                                       aria-expanded="false"><i className="fa fa-wrench"/></a>
                                    <ul className="dropdown-menu" role="menu">
                                        {React.Children.map(this.props.children, child => <li>{child}</li>)}
                                    </ul>
                                </li>
                            }
                            <li><a className="close-link"><i className="fa fa-close"/></a>
                            </li>
                        </ul>
                        <Clearfix/>
                    </div>
                    <div className="x_content">
                        {/* The chart will be generated here after the component is mounted (this.componentDidMount). */}
                        <canvas ref={data => {
                            this.data = data
                        }}/>
                    </div>
                </div>
            </div>
        );
    }
}

// Row renders a row component of charts only if there is any chart.
class Row extends Component {
    render = () => isNullOrUndefined(this.props.children) || <div className="row"> {this.props.children} </div>;
}

// PageContent renders a component for the page content.
class PageContent extends Component {
    render = () => <div className="right_col" role="main">
        <div className="">
            <div className="page-title">
                <div className="title_left">
                    <h3>Go Ethereum Dashboard
                        <small>Statistics</small>
                    </h3>
                </div>
            </div>
            <Clearfix/>
            <Row>
                <ChartComponent className="col-md-6 col-sm-6 col-xs-12" text="Memory usage system/memory/inuse" type="line" data={this.props.charts.memory}>
                    <a href="#">Settings 1</a>
                    <a href="#">Settings 2</a>
                </ChartComponent>
                <ChartComponent className="col-md-6 col-sm-6 col-xs-12" text="Inbound traffic p2p/InboundTraffic" type="line" data={this.props.charts.traffic}/>
            </Row>
            <Clearfix/>
        </div>
    </div>;
}

// Footer renders a footer.
let Footer = () => <footer>
    <div className="pull-right">
        Gentelella - Bootstrap Admin Template by <a href="https://colorlib.com">Colorlib</a>
    </div>
    <Clearfix/>
</footer>;

const MEMORY_SAMPLE_LIMIT = 200; // Maximum number of memory data samples.
const TRAFFIC_SAMPLE_LIMIT = 200; // Maximum number of traffic data samples.

// Dashboard renders a full dashboard component, which makes connection with the server and waits for messages.
// When there is a message, correspondingly updates the page's content.
class Dashboard extends Component {
    constructor(props) {
        super(props);
        this.state = {
            charts: {
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

    updateCharts = msg => {
        let memory = this.state.charts.memory;
        let traffic = this.state.charts.traffic;

        // Fill the dashboard with the past data. metrics is set only in the first msg, after the connection is established.
        if (msg.metrics !== undefined) {
            // Clear the arrays to prevent data confusion with the previous connection.
            memory.labels = [];
            traffic.labels = [];
            memory.datasets[0].data = [];
            traffic.datasets[0].data = [];

            let mem = msg.metrics.memory;
            let traff = msg.metrics.processor; // TODO (kurkomisi): !!!

            // Put the past data to the beginning of the arrays. This prevents confusion with the next msg data, which goes to the end.
            for (let i = mem.length - 1; i >= 0 && MEMORY_SAMPLE_LIMIT > memory.labels.length; --i) {
                memory.labels.unshift(mem[i].time.substring(mem[i].time.length - 5));
                traffic.labels.unshift(mem[i].time.substring(mem[i].time.length - 5));
                memory.datasets[0].data.unshift(mem[i].value);
                traffic.datasets[0].data.unshift(traff[i].value);
            }

            this.setState({charts: {memory: memory, traffic: traffic,}}); // Update the components.
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

            this.setState({charts: {memory: memory, traffic: traffic,}}); // Update the components.
        }
    };

    reconnect = () => {
        let server = new WebSocket("ws://" + location.host + "/api");
        let that = this;

        server.onmessage = event => {
            let msg = JSON.parse(event.data);
            if (isNullOrUndefined(msg)) {
                return;
            }
            that.updateCharts(msg);
        };

        server.onclose = () => setTimeout(that.reconnect, 3000);
    };

    componentDidMount = () => this.reconnect();

    render = () => <div className="container body">
        <div className="main_container">
            <SideBar/>
            <TopNavigation/>
            <PageContent charts={this.state.charts}/>
            <Footer/>
        </div>
    </div>;
}