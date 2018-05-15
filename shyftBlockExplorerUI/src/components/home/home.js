import React from "react";
import classes from './home.css';
import { Link } from 'react-router-dom'

const home = props => {

    return (
       <div className={classes.Home}>
            <span>THIS IS A WIP</span>
           <div className={classes.Transactions}>
               <Link to="/transactions"><button>Transactions</button></Link>
           </div>
           <div className={classes.Transactions}>
               <Link to="/blocks"><button>Blocks</button></Link>
           </div>
       </div>
    );
};

export default home;
