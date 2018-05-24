import React, { Component } from 'react';
import MinedBlockTable from './blocksMinedTable';
import Loading from '../../UI materials/loading'
import classes from './table.css';

class BlocksMinedTable extends Component {
    constructor(props) {
        super(props);
        this.state = {
            data: []
        };
    }

    render() {
        const table = this.props.data.map((data, i) => {
            const conversion = data.Rewards / 10000000000000000000;
            return <MinedBlockTable
                key={`${data.Hash}${i}`}
                Hash={data.Hash}
                Number={data.Number}
                Coinbase={data.Coinbase}
                Age={data.Age}
                GasUsed={data.GasUsed}
                GasLimit={data.GasLimit}
                UncleCount={data.UncleCount}
                TxCount={data.TxCount}
                Reward={conversion}
                detailBlockHandler={this.props.detailBlockHandler}
                getBlocksMined={this.props.getBlocksMined}
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
export default BlocksMinedTable;
