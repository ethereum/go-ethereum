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
import BlockTxs from "../components/table/transactions/blockTx";

///**BLOCKS**///
import BlocksRow from '../components/table/blocks/blockRows';
import DetailBlockTable from '../components/table/blocks/blocksDetailsRow';
import BlockDetailHeader from "../components/nav/blockHeaders/blockDetailHeader";
import BlockHeader from "../components/nav/blockHeaders/blockHeader";
import BlocksMinedTable from "../components/table/blocks/blocksMined";
import BlockCoinbaseHeader from "../components/nav/blockHeaders/blockCoinbaseHeader";
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
        blocksMined: [],
        blockTransactions: [],
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
    };

    detailTransactionHandler = async(txHash) => {
        try {
            const response = await axios.get(`http://localhost:8080/api/get_transaction/${txHash}`);
            await this.setState({ transactionDetailData: response.data })
        }
        catch(error) {
            console.log(error)
        }
    };

    detailAccountHandler = async(addr) => {
        try {
            const response = await axios.get(`http://localhost:8080/api/get_account_txs/${addr}`)
            await this.setState({ accountDetailData: response.data, reqAccount: addr })
        }
        catch(error) {
            console.log(error)
        }
    };

    getBlockTransactions = async(blockNumber) => {
        try {
            const response = await axios.get(`http://localhost:8080/api/get_all_transactions_from_block/${blockNumber}`)
            await this.setState({ blockTransactions: response.data })
        }
        catch(error) {
            console.log(error)
        }
    };

    getBlocksMined = async(coinbase) => {
        console.log("THIS RAN")
        try {
            const response = await axios.get(`http://localhost:8080/api/get_blocks_mined/${coinbase}`)
            console.log(response.data);
            await this.setState({ blocksMined: response.data })
        }
        catch(error) {
            console.log(error)
        }
    };

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
                  <TransactionRow
                      getBlockTransactions={this.getBlockTransactions}
                      detailTransactionHandler={this.detailTransactionHandler}
                      detailAccountHandler={this.detailAccountHandler}/>
              </div>}
          />

          <Route path="/blocks" exact render={({match}) =>
              <div>
                  <BlockHeader/>
                  <BlocksRow
                      getBlocksMined={this.getBlocksMined}
                      getBlockTransactions={this.getBlockTransactions}
                      detailBlockHandler={this.detailBlockHandler}/>
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
                      getBlockTransactions={this.getBlockTransactions}
                      data={this.state.blockDetailData}/>
              </div>}
          />

          <Route path="/block/transactions" exact render={({match}) =>
              <div>
                  <BlockDetailHeader
                      blockNumber={this.state.blockTransactions.BlockNumber}/>
                  <BlockTxs
                      data={this.state.blockTransactions}
                      getBlockTransactions={this.getBlockTransactions}
                      detailTransactionHandler={this.detailTransactionHandler}
                      detailAccountHandler={this.detailAccountHandler}/>

              </div>}
          />

          <Route path="/mined/blocks" exact render={({match}) =>
              <div>
                  <BlockCoinbaseHeader
                      data={this.state.blocksMined}/>
                  <BlocksMinedTable
                      getBlockTransactions={this.getBlockTransactions}
                      getBlocksMined={this.getBlocksMined}
                      data={this.state.blocksMined}/>
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
