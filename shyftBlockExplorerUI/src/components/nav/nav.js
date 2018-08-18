import React from "react";
import classes from './nav.css';
import { Link } from 'react-router-dom'

class navBar extends React.Component  {

  constructor(props) {
    super(props);
    this.state = {
      active: "blocks"
    };
  }

  updateActive = (page) => {
    this.setState({ active: page  });
  }

  render() {
    return (
      <div>
        <div className={classes.navHeader}>
          <h5 className={classes.headerText}> Block Explorer </h5>
          <div className={classes.buttonContainer}>
            <Link to="/blocks">  
              <button 
                className={this.state.active === "blocks" ? classes.btnActive : classes.btn}
                onClick={ ()=>this.updateActive("blocks") } > 
                BLOCKS 
              </button> 
            </Link>   
            <Link to="/transactions">  
              <button 
                className={this.state.active === "transactions" ? classes.btnActive : classes.btn}
                onClick={ ()=>this.updateActive("transactions") }> 
                TRANSACTIONS 
              </button>
            </Link>   
            <button 
              className={this.state.active === "internal" ? classes.btnActive : classes.btn}
              onClick={ ()=> this.updateActive("internal") }> 
              INTERNAL TX 
            </button>      
            <Link to="/accounts">  
              <button 
               className={this.state.active === "accounts" ? classes.btnActive : classes.btn}
                onClick={ ()=>this.updateActive("accounts") }>                 
                ACCOUNTS 
              </button>
            </Link>   
          </div>
        </div>
      </div>
    )
  };
};

export default navBar;
