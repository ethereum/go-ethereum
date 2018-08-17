import React from "react";
import classes from './home.css';
import { Link } from 'react-router-dom'

const home = props => {

    const combinedClasses = ["btn", "btn-primary", classes.BlockButton]
    const conjoinecdClasses = ["btn", "btn-primary", classes.AccountButton]

    return (
        <div></div>
    //    <div className={classes.Home}>
    //         <span className={classes.Greeting}>THIS IS A WIP</span>
    //        <div>
    //        <div className={classes.Transactions}>
    //            <Link to="/transactions"><button className="btn btn-primary">Transactions</button></Link>
    //        </div>
    //        <div className={classes.Blocks}>
    //            <Link to="/blocks"><button className={combinedClasses.join(" ")}>Blocks</button></Link>
    //        </div>
    //        <div className={classes.Accounts}>
    //            <Link to="/accounts"><button className={conjoinecdClasses.join(" ")}>Accounts</button></Link>
    //        </div>
    //        </div>
    //    </div>
    );
};

export default home;
