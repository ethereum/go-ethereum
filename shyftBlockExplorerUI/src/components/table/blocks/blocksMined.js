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
                AgeGet={data.AgeGet}
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
                <thead>
                <tr>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>Height</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>Block Hash</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>Age</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>Txn</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>Uncles</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>Coinbase</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>GasUsed</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>GasLimit</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>Avg.GasPrice</th>
                    <th scope="col" style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e"}}>Reward</th>
                </tr>
                </thead>
                {table}
            </table>
        );
    }
}
export default BlocksMinedTable;
