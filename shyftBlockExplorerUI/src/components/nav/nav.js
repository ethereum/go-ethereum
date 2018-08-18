import React from "react";
import classes from './nav.css';
import { Link } from 'react-router-dom'

class navBar extends React.Component  {

  constructor(props) {
    super(props);
    this.state = {
      selected: ""
    };
  }

  // updateSelected = (page) => {
  //   this.setState({ selected });
  // }

  render() {
    return (
      <div>
        <div className={classes.navHeader}>
          <h5 className={classes.headerText}> Block Explorer </h5>
          <div className={classes.buttonContainer}>

            <Link to="/blocks">  
              <button className={classes.btn} > BLOCKS </button> 
            </Link>   

            <Link to="/transactions">  
              <button className={classes.btn} > TRANSACTIONS </button>
            </Link>   

            <button className={classes.btn}> INTERNAL TX </button>
      
            <Link to="/accounts">  
              <button className={classes.btn}> ACCOUNTS </button>
            </Link>   

          </div>
        </div>
      </div>

     /* <nav className="navbar navbar-light justify-content-between">
      <a className={combinedClasses.join(" ")}>Block Explorer Test UI</a>
     <form className="form-inline">
        <input
          className="form-control mr-sm-2"
          type="search"
          placeholder="Search"ls

          aria-label="Search"
        />
        <button className="btn btn-outline-success my-2 my-sm-0" type="submit">
          Search
        </button>
      </form> 
    </nav>*/
    )
  };
};

export default navBar;
