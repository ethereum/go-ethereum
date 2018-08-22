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
            const response = await axios.get("http://localhost:8080/api/get_all_blocks");
            await this.setState({data: response.data});
        } catch (err) {
            console.log(err);
        }
    }

    render() {
        const table = this.state.data.map((data, i) => {
            const conversion = data.Rewards / 10000000000000000000;
            return <BlockTable
                key={`${data.TxHash}${i}`}
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
            <div>
                <table className={combinedClasses.join(' ')}>
                    <thead>
                        <tr>
                            <th scope="col" className={classes.thItem}> Height </th>
                            <th scope="col" className={classes.thItem}> Block Hash </th>
                            <th scope="col" className={classes.thItem}> Age </th>
                            <th scope="col" className={classes.thItem}> Txn </th>
                            <th scope="col" className={classes.thItem}> Uncles </th>
                            <th scope="col" className={classes.thItem}> Coinbase </th>
                            <th scope="col" className={classes.thItem}> GasUsed </th>
                            <th scope="col" className={classes.thItem}> GasLimit </th>
                            <th scope="col" className={classes.thItem}> Avg.GasPrice </th>
                            <th scope="col" className={classes.thItem}> Reward </th>
                        </tr>
                    </thead>
                    {table}
                </table>
            </div>
        );
    }
}
export default BlocksTable;
