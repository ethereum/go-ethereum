import React, { Component } from 'react';
import classes from './table.css';

class DetailTransactionTable extends Component {

    render() {
        let data = this.props.data
        let combinedClasses = ['responsive-table', classes.table];
        return (
            <table className={combinedClasses.join(' ')}>
                <tbody>
                <tr>
                    <th scope="col">TxHash:</th>
                    <td>{data.TxHash}</td>
                </tr>
                <tr>
                    <th scope="col">TxReceipt Status:</th>
                    <td>{data.Status}</td>
                </tr>
                <tr>
                    <th scope="col">Block Height:</th>
                    <td>{data.BlockNumber}</td>
                </tr>
                <tr>
                    <th scope="col">TimeStamp:</th>
                    <td>{data.Age}</td>
                </tr>
                <tr>
                    <th scope="col">From:</th>
                    <td>{data.From}</td>
                </tr>
                <tr>
                    <th scope="col">To:</th>
                    <td>{ `${data.IsContract}` ? `${data.To} (Contract)` : `${data.To}` }</td>
                </tr>
                <tr>
                    <th scope="col">Value:</th>
                    <td>{data.Amount}</td>
                </tr>
                <tr>
                    <th scope="col">Gas Limit:</th>
                    <td>{data.GasLimit}</td>
                </tr>
                <tr>
                    <th scope="col">Gas Used By Txn:</th>
                    <td>{data.Gas}</td>
                </tr>
                <tr>
                    <th scope="col">Gas Price:</th>
                    <td>{data.GasPrice}</td>
                </tr>
                <tr>
                    <th scope="col">Actual Tx Cost/Fee:</th>
                    <td>{data.Cost}</td>
                </tr>
                <tr>
                    <th scope="col">Nonce:</th>
                    <td>{data.Nonce}</td>
                </tr>
                <tr>
                    <th scope="col">Input Data:</th>
                    <td>{data.Data}</td>
                </tr>
                </tbody>
            </table>
        );
    }
}
export default DetailTransactionTable;
