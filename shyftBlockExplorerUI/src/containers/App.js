import React, { Component } from "react";
import axios from 'axios';
import Nav from "../components/nav/nav";
import { BrowserRouter, Route } from 'react-router-dom'

///**LANDING PAGE**///
import Home from '../components/home/home';

///**TRANSACTIONS**///
import TransactionRow from '../components/table/transactions/transactionRow';
import TransactionHeader from "../components/nav/transactionHeader/transactionHeader";
import TransactionDetailHeader from "../components/nav/transactionHeader/transactionDetailHeader";
import DetailTransactionTable from "../components/table/transactions/transactionDetailsRow";
///**BLOCKS**///
import BlocksRow from '../components/table/blocks/blockRows';
import DetailBlockTable from '../components/table/blocks/blocksDetailsRow';
import BlockDetailHeader from "../components/nav/blockHeaders/blockDetailHeader";
import BlockHeader from "../components/nav/blockHeaders/blockHeader";

///**ACCOUNTS**///
import AccountsRow from '../components/table/accounts/accountRows';
import DetailAccountsTable from "../components/table/accounts/detailAccountsRow";
import AccountHeader from "../components/nav/accountHeaders/accountHeader";
import AccountDetailHeader from "../components/nav/accountHeaders/accountDetailHeader";


class App extends Component {
  constructor(props) {
    super(props);
    this.state = {
        blockDetailData: [],
        transactionDetailData: [],
        accountDetailData: [],
        reqAccount: ''
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
        try {
            const response = await axios.get(`http://localhost:8080/api/get_transaction/${txHash}`);
            await this.setState({ transactionDetailData: response.data })
        }
        catch(error) {
            console.log(error)
        }
    }

    detailAccountHandler = async(addr) => {
        try {
            const response = await axios.get(`http://localhost:8080/api/get_account_txs/${addr}`)
            await this.setState({ accountDetailData: response.data, reqAccount: addr })
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

          <Route path="/accounts" exact render={({match}) =>
              <div>
                  <AccountHeader/>
                  <AccountsRow detailAccountHandler={this.detailAccountHandler}/>
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
                  <DetailBlockTable
                      data={this.state.blockDetailData}/>
              </div>}
          />

          <Route path="/account/detail" exact render={({match}) =>
              <div>
                  <AccountDetailHeader
                      addr={this.state.reqAccount}/>
                  <DetailAccountsTable
                      transactionDetailHandler={this.detailTransactionHandler}
                      addr={this.state.reqAccount}
                      data={this.state.accountDetailData}/>
              </div>}
          />
      </div>
        </BrowserRouter>
    );
  }
}

export default App;
