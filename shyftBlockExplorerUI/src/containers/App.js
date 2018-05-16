import React, { Component } from "react";
import axios from 'axios';
import Nav from "../components/nav/nav";
import { BrowserRouter, Route, Link } from 'react-router-dom'
import TransactionRow from '../components/table/transactions/transactionRow';
import BlocksRow from '../components/table/blocks/blockRows';
import DetailBlockHeader from '../components/table/blocks/blocksDetailsRow';
import TransactionHeader from "../components/nav/transactionHeader/transactionHeader";
import TransactionDetailHeader from "../components/nav/transactionHeader/transactionDetailHeader";
import BlockDetailHeader from "../components/nav/blockHeaders/blockDetailHeader";
import BlockHeader from "../components/nav/blockHeaders/blockHeader";
import Home from '../components/home/home';
import DetailTransactionTable from "../components/table/transactions/transactionDetailsRow";

class App extends Component {
  constructor(props) {
    super(props);
    this.state = {
        blockDetailData: [],
        transactionDetailData: []
    };
  }

    detailBlockHandler = async(blockNumber) => {
        try {
            const response = await axios.get(`http://localhost:8080/api/get_block/${blockNumber}`)
            await this.setState({ blockDetailData: response.data })
        }
        catch(error) {
           console.log(error)
        }
    }

    detailTransactionHandler = async(txHash) => {
      console.log("THIS RAN")
        try {
            const response = await axios.get(`http://localhost:8080/api/get_transaction/${txHash}`)
            console.log("THIS IS RESPONSE", response)
            await this.setState({ transactionDetailData: response.data })
        }
        catch(error) {
            console.log(error)
        }
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
                  <TransactionRow detailTransactionHandler={this.detailTransactionHandler}/>
              </div>}
          />

          <Route path="/blocks" exact render={({match}) =>
              <div>
                  <BlockHeader/>
                  <BlocksRow detailBlockHandler={this.detailBlockHandler}/>
              </div>}
          />

          <Route path="/transaction/details" exact render={({match}) =>
              <div>
                  <TransactionDetailHeader
                    txHash={this.state.transactionDetailData.TxHash}/>
                  <DetailTransactionTable
                    data={this.state.transactionDetailData}/>
              </div>}
          />

          <Route path="/blocks/detail" exact render={({match}) =>
              <div>
                  <BlockDetailHeader
                      blockNumber={this.state.blockDetailData.Number}/>
                  <DetailBlockHeader
                      data={this.state.blockDetailData}/>
              </div>}
          />


      </div>
        </BrowserRouter>
    );
  }
}

export default App;
