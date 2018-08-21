import React, { Component } from "react";
import axios from 'axios';
import Nav from "../components/nav/nav";
import { BrowserRouter, Route } from 'react-router-dom'

import { Link } from 'react-router-dom'

import Modal from 'react-bootstrap/lib/Modal';
import Button from 'react-bootstrap/lib/Button';
import Grid from 'react-bootstrap/lib/Grid';
import Col from 'react-bootstrap/lib/Col';
import Row from 'react-bootstrap/lib/Row';

///**LANDING PAGE**///
import Home from '../components/home/home';

///**TRANSACTIONS**///
import TransactionRow from '../components/table/transactions/transactionRow';
import TransactionDetailHeader from "../components/nav/transactionHeader/transactionDetailHeader";
import DetailTransactionTable from "../components/table/transactions/transactionDetailsRow";
import BlockTxs from "../components/table/transactions/blockTx";

///**BLOCKS**///
import BlocksRow from '../components/table/blocks/blockRows';
import DetailBlockTable from '../components/table/blocks/blocksDetailsRow';
import BlockDetailHeader from "../components/nav/blockHeaders/blockDetailHeader";
import BlocksMinedTable from "../components/table/blocks/blocksMined";
import BlockCoinbaseHeader from "../components/nav/blockHeaders/blockCoinbaseHeader";

///**ACCOUNTS**///
import AccountsRow from '../components/table/accounts/accountRows';
import DetailAccountsTable from "../components/table/accounts/detailAccountsRow";
import AccountDetailHeader from "../components/nav/accountHeaders/accountDetailHeader";

import classes from './App.css';


class App extends Component {
  constructor(props) {
    super(props);
    this.state = {
        blockDetailData: [],
        transactionDetailData: [],
        accountDetailData: [],
        blocksMined: [],
        blockTransactions: [],
        reqAccount: '',
        overlayTriggered: false,
        overlayContent: "",
    };
  }

    detailBlockHandler = async(blockNumber) => {
        try {
            const response = await axios.get(`http://localhost:8080/api/get_block/${blockNumber}`)
            await this.setState({ blockDetailData: response.data, overlayTriggered: true, overlayContent: "block" })
        }
        catch(error) {
           console.log(error)
        }
    };

    detailTransactionHandler = async(txHash) => {
        try {
            const response = await axios.get(`http://localhost:8080/api/get_transaction/${txHash}`);
            await this.setState({ transactionDetailData: response.data, overlayTriggered: true, overlayContent: "transaction" })
        }
        catch(error) {
            console.log(error)
        }
    };

    detailAccountHandler = async(addr) => {

        console.log("[DETAIL ACCOUNT Handler]");

        try {
            const response = await axios.get(`http://localhost:8080/api/get_account_txs/${addr}`)
            await this.setState({ accountDetailData: response.data, reqAccount: addr, overlayTriggered: true, overlayContent: "account" })
        }
        catch(error) {
            console.log(error)
        }
    };

    getBlockTransactions = async(blockNumber) => {
        try {
            const response = await axios.get(`http://localhost:8080/api/get_all_transactions_from_block/${blockNumber}`)
            await this.setState({ blockTransactions: response.data, reqBlockNum: blockNumber })
        }
        catch(error) {
            console.log(error)
        }
    };

    getBlocksMined = async(coinbase) => {
        try {
            const response = await axios.get(`http://localhost:8080/api/get_blocks_mined/${coinbase}`)
            await this.setState({ blocksMined: response.data, reqCoinbase: coinbase })
        }
        catch(error) {
            console.log(error)
        }
    };

    hideOverlay = () => {
        this.setState({overlayTriggered: false})
    };

    getBlockData = (data, page) => {
        let components = [];
        console.log("ACCOUNT DATA", data)   
        let dataEntry;
        
        if(page === "account") {
            dataEntry = data[0];
        } else { 
            dataEntry = data;
        }
        
        for (let key in dataEntry) {
            let value = dataEntry[key]
            // console.log(key, value)            
            components.push( 
                <Grid>
                    <Row className="show-grid">
                        <Col xs={3}  md={3}  style={{fontSize:'6.5pt', color: '#565656', paddingTop: '10pt', fontFamily: 'Open Sans, sans-serif'}}>
                            {key}
                        </Col>
                        <Col xs={9}  md={9}  style={{fontSize:'6.5pt', color: '#565656', paddingTop: '10pt', fontFamily: 'Open Sans, sans-serif'}}>
                            {value}
                        </Col>
                    </Row>
                </Grid>
            );          
        }

        return components;
    }

    renderOverlay = () => {

        const page = this.state.overlayContent;
        let data, title;
        let keys = [];
        let values = [];

        switch (page) {
            case "block" : 
                title = "BLOCK OVERVIEW";
                data = this.state.blockDetailData;
            break; 
            case "transaction" :
                title = "TRANSACTION OVERVIEW";
                data =  this.state.transactionDetailData;
            break;
            case "account" : 
                title = "ACCOUNT OVERVIEW";
                data =  this.state.accountDetailData;
            break;
            default: console.log("error");
        }

        return ( 
            <div className="static-modal">
                <Modal.Dialog>
                    <Modal.Header>
                        <Modal.Title style={{color: '#593c83', fontSize: '8pt', letterSpacing: '2pt',  fontFamily: 'Open Sans, sans-serif' }}> {title} </Modal.Title>
                    </Modal.Header>
                    <Modal.Body>
                        <ul>
                        {this.getBlockData(data, page)}         
                        </ul>       
                    </Modal.Body>
                    <Modal.Footer>
                    <Button onClick={this.hideOverlay}>Close</Button>                   
                    </Modal.Footer>
                </Modal.Dialog>
            </div>
        );
    };

  render() {
    return (
        <BrowserRouter>
            <div>
                
                {
                    this.state.overlayTriggered ? this.renderOverlay() : null
                }
        
                <div style={ this.state.overlayTriggered ? {backgroundColor:"#f7f8f9", paddingBottom:"5%", opacity:0.5, zIndex: -10000, height: '100%'}  : {backgroundColor:"#f7f8f9", paddingBottom:"5%", height: '100%' }  }>
                    <Nav />
        
                    <Route path="/" exact render={({ match }) =>
                        <Home/> 
                        }
                    />

                    <Route path="/transactions" render={({match}) =>
                        <div>                  
                            <TransactionRow
                                getBlockTransactions={this.getBlockTransactions}
                                detailTransactionHandler={this.detailTransactionHandler}
                                detailAccountHandler={this.detailAccountHandler}/>
                        </div>
                        }
                    />

                    <Route path="/blocks" exact render={({match}) =>
                        <div>                        
                            <BlocksRow
                                getBlocksMined={this.getBlocksMined}
                                getBlockTransactions={this.getBlockTransactions}
                                detailBlockHandler={this.detailBlockHandler}/>                        
                        </div>
                        }
                    />

                    <Route path="/accounts" exact render={({match}) =>
                        <div>         
                            <AccountsRow detailAccountHandler={this.detailAccountHandler}/>
                        </div>
                        }
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
                            blockNumber={this.state.reqBlockNum}/>
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
                            coinbase={this.state.reqCoinbase}/>
                        <BlocksMinedTable
                            getBlockTransactions={this.getBlockTransactions}
                            getBlocksMined={this.getBlocksMined}
                            data={this.state.blocksMined}/>
                    </div>}
                    />

                    <Route path="/account/detail" exact render={({match}) =>
                    <div>
                        {/* <AccountDetailHeader
                            addr={this.state.reqAccount}
                            data={this.state.accountDetailData}/> */}
                        <DetailAccountsTable
                            transactionDetailHandler={this.detailTransactionHandler}
                            addr={this.state.reqAccount}
                            data={this.state.accountDetailData}/>
                    </div>}
                    />
            </div>
        </div>
    </BrowserRouter>
    );
  }
}

export default App;
