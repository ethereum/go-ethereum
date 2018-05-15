import React from "react";
import classes from './nav.css';

const navBar = props => {
    let combinedClasses = ["navbar-brand", classes.TopBar];
  return (
    <nav className="navbar navbar-light justify-content-between">
      <a className={combinedClasses.join(" ")}>Block Explorer Test UI</a>
      {/* <form className="form-inline">
        <input
          className="form-control mr-sm-2"
          type="search"
          placeholder="Search"
          aria-label="Search"
        />
        <button className="btn btn-outline-success my-2 my-sm-0" type="submit">
          Search
        </button>
      </form> */}
    </nav>
  );
};

export default navBar;
