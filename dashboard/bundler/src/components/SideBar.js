import Component from 'component';
import {isNullOrUndefined, mapChildren, Clearfix} from "./Common";

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
                {mapChildren(this.props.children, child => <li>{child}</li>)}
            </ul>
        }
    </li>;
}

// Menu renders a menu component.
const Menu = () => <ul className="nav side-menu">
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

// SideBar renders a sidebar component.
export const SideBar = () => <div className="col-md-3 left_col">
    <div className="left_col scroll-view">
        <div className="navbar nav_title" style={{border: 0}}>
            <a href="dashboard.html" className="site_title"><i className="fa fa-paw"/>
                <span>Go Ethereum Dashboard</span></a>
        </div>
        <Clearfix/>
        <div id="sidebar-menu" className="main_menu_side hidden-print main_menu">
            <div className="menu_section">
                <Menu/>
            </div>
        </div>
    </div>
</div>;