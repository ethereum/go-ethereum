import React, { Component } from 'react';
import classes from './table.css';

class DetailBlockTable extends Component {

    render() {
        let data = this.props.data
        let combinedClasses = ['responsive-table', classes.table];
        return (
            <table className={combinedClasses.join(' ')}>
                <tr>
                    <th scope="col">Height:</th>
                    <td>{data.Number}</td>
                </tr>
                <tr>
                    <th scope="col">Age:</th>
                    <td>{data.Age}</td>
                </tr>
                <tr>
                    <th scope="col">Txn:</th>
                    <td>{data.TxCount} transactions</td>
                </tr>
                <tr>
                    <th scope="col">Block Hash:</th>
                    <td>{data.Hash}</td>
                </tr>
                <tr>
                    <th scope="col">Parent Hash:</th>
                    <td>{data.ParentHash}</td>
                </tr>
                <tr>
                    <th scope="col">Uncle Hash:</th>
                    <td>{data.UncleHash}</td>
                </tr>
                <tr>
                    <th scope="col">Uncle Count:</th>
                    <td>{data.UncleCount}</td>
                </tr>
                <tr>
                    <th scope="col">Coinbase:</th>
                    <td>{data.Coinbase}</td>
                </tr>
                <tr>
                    <th scope="col">Difficulty:</th>
                    <td>{data.Difficulty}</td>
                </tr>
                <tr>
                    <th scope="col">GasUsed:</th>
                    <td>{data.GasUsed}</td>
                </tr>
                <tr>
                    <th scope="col">Size:</th>
                    <td>{data.Size}ytes</td>
                </tr>
                <tr>
                    <th scope="col">GasUsed:</th>
                    <td>{data.GasUsed}</td>
                </tr>
                <tr>
                    <th scope="col">GasLimit:</th>
                    <td>{data.GasLimit}</td>
                </tr>
                <tr>
                    <th scope="col">Nonce:</th>
                    <td>{data.Nonce}</td>
                </tr>
                <tr>
                    <th scope="col">Reward:</th>
                    <td>TBD</td>
                </tr>
            </table>
        );
    }
}
export default DetailBlockTable;
