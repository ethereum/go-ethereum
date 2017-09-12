import Component from 'component';
import {isNullOrUndefined, mapChildren, Clearfix} from "./Common";

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

        return <div className={this.props.className}>
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
                                    {mapChildren(this.props.children, child => <li>{child}</li>)}
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
        </div>;
    }
}

// Row renders a row component of charts only if there is any chart.
class Row extends Component {
    render = () => isNullOrUndefined(this.props.children) || <div className="row"> {this.props.children} </div>;
}

// PageContent renders a component for the page content.
export default class PageContent extends Component {
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
                <ChartComponent className="col-md-6 col-sm-6 col-xs-12" text="Memory usage system/memory/inuse"
                                type="line" data={this.props.charts.memory}>
                    <a href="#">Settings 1</a>
                    <a href="#">Settings 2</a>
                </ChartComponent>
                <ChartComponent className="col-md-6 col-sm-6 col-xs-12" text="Inbound traffic p2p/InboundTraffic"
                                type="line" data={this.props.charts.traffic}/>
            </Row>
            <Clearfix/>
        </div>
    </div>;
}