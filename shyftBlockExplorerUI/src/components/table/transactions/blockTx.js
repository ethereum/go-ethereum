import React, { Component } from 'react';
import TransactionsTable from './transactionTable';
import classes from './table.css';

class BlockTransactionTable extends Component {
    constructor(props) {
        super(props);
        this.state = {
            data: []
        };
    }

    render() {
        const table = this.props.data.map((data, i) => {
            const conversion = data.Cost / 10000000000000000000;
            return <TransactionsTable
                key={`${data.TxHash}${i}`}
                age={data.Age}
                txHash={data.TxHash}
                blockNumber={data.BlockNumber}
                to={data.To}
                from={data.From}
                value={data.Amount}
                cost={conversion}
                getBlockTransactions={this.props.getBlockTransactions}
                detailTransactionHandler={this.props.detailTransactionHandler}
                detailAccountHandler={this.props.detailAccountHandler}
            />
        })

        let combinedClasses = ['responsive-table', classes.table];
        return (
            <table key={this.props.data.TxHash} className={combinedClasses.join(' ')}>
                <thead>
                <tr>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>TxHash</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>Block</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>Age</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>From</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}></th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>To</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>Value</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>TxFee</th>
                </tr>
                </thead>
                {table}
            </table>
        );
    }
}
export default BlockTransactionTable;
