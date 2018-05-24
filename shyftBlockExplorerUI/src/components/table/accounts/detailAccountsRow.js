import React, { Component } from 'react';
import DetailAccountsTable from './detailAccountsTable';
import ErrorHandler from "./errorMessage";
import classes from './table.css';

class AccountTransactionTable extends Component {
    render() {
        let table;
        if(this.props.data.length <= 1) {
           return <ErrorHandler />
        }else {
            table = this.props.data.map((data, i) => {
                console.log(this.props.data)
                // const total = this.props.data
                //     .map(num => num.Balance / 10000000000000000000)
                //     .reduce((acc, cur) => acc + cur ,0);
                const conversion = data.Cost / 10000000000000000000;
                return <DetailAccountsTable
                    key={`${data.TxHash}${i}`}
                    age={data.Age}
                    txHash={data.TxHash}
                    blockNumber={data.BlockNumber}
                    to={data.To}
                    from={data.From}
                    value={data.Amount}
                    cost={conversion}
                    addr={this.props.addr}
                    detailTransactionHandler={this.props.transactionDetailHandler}
                />
            })
        }

        let combinedClasses = ['responsive-table', classes.table];
        return (
            <table key={this.props.data.TxHash} className={combinedClasses.join(' ')}>
                <thead className={classes.tHead}>
                <tr>
                    <th scope="col">TxHash</th>
                    <th scope="col">Block</th>
                    <th scope="col">Age</th>
                    <th scope="col">From</th>
                    <th scope="col"></th>
                    <th scope="col">To</th>
                    <th scope="col">Value</th>
                    <th scope="col">TxFee</th>
                </tr>
                </thead>
                {table}
            </table>
        );
    }
}
export default AccountTransactionTable;
