import React, { Component } from "react";
import Nav from "../components/nav/nav";
import axios from "axios";
import BlockInfoTable from "../components/table/transactions/transactionTable";
import { BrowserRouter, Route, Link } from 'react-router-dom'
import TransactionRow from '../components/table/transactions/transactionRow';
import BlocksRow from '../components/table/blocks/blockRows';
import TransactionHeader from "../components/nav/transactionHeader";
import BlockHeader from "../components/nav/blockHeader";
import Home from '../components/home/home';
import classes from "./App.css";

class App extends Component {
  constructor(props) {
    super(props);
    this.state = {
      data: []
    };
  }

  render() {
    return (
        <BrowserRouter>
      <div className="container">
        <Nav />

          <Route path="/" exact render={({ match }) =>
              <Home/>}
          />

          <Route path="/transactions" render={({match}) =>
              <div>
                  <TransactionHeader />
              <TransactionRow />
              </div>}
          />

          <Route path="/blocks" render={({match}) =>
              <div>
                  <BlockHeader/>
                  <BlocksRow />
              </div>}
          />


      </div>
        </BrowserRouter>
    );
  }
}

export default App;
