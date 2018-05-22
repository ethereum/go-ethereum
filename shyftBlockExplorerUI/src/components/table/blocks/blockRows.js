import React, { Component } from 'react';
import BlockTable from './blockTable';
import Loading from '../../UI materials/loading'
import classes from './table.css';
import axios from "axios/index";

class BlocksTable extends Component {
    constructor(props) {
        super(props);
        this.state = {
            data: []
        };
    }

    async componentDidMount() {
        try {
            const response = await axios.get("http://localhost:8080/api/get_all_blocks")
            await this.setState({data: response.data});
        } catch (err) {
            console.log(err);
        }
    }


    render() {
        const table = this.state.data.map((data, i) => {
            return <BlockTable
                key={`${data.TxHash}${i}`}
                Hash={data.Hash}
                Number={data.Number}
                Coinbase={data.Coinbase}
                Age={data.Age}
                GasUsed={data.GasUsed}
                GasLimit={data.GasLimit}
                UncleCount={data.UncleCount}
                TxCount={data.TxCount}
                detailBlockHandler={this.props.detailBlockHandler}
            />
        });

        let combinedClasses = ['responsive-table', classes.table];
        return (
            <table className={combinedClasses.join(' ')}>
                <thead className={classes.tHead}>
                <tr>
                    <th scope="col">Height</th>
                    <th scope="col">Block Hash</th>
                    <th scope="col">Age</th>
                    <th scope="col">Txn</th>
                    <th scope="col">Uncles</th>
                    <th scope="col">Coinbase</th>
                    <th scope="col">GasUsed</th>
                    <th scope="col">GasLimit</th>
                    <th scope="col">Avg.GasPrice</th>
                    <th scope="col">Reward</th>
                </tr>
                </thead>
                        {table}
            </table>
        );
    }
}
export default BlocksTable;
