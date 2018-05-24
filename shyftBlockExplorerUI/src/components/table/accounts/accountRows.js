import React, { Component } from 'react';
import AccountsTable from './accountsTable';
import classes from './accounts.css';
import axios from "axios/index";

class AccountTable extends Component {
    constructor(props) {
        super(props);
        this.state = {
            data: []
        };
    }

    async componentDidMount() {
        try {
            const response = await axios.get("http://localhost:8080/api/get_all_accounts")
            await this.setState({data: response.data});
        } catch (err) {
            console.log(err);
        }
    }

    render() {

        let startNum = 1;
        const sorted = [...this.state.data];
            sorted.sort((a, b) => Number(a.Balance) > Number(b.Balance));
            console.log(sorted);
        const table = sorted.reverse().map((data, i) => {
            const conversion = Number(data.Balance) / 10000000000000000000;
            const total = sorted
                .map(num => Number(num.Balance) / 10000000000000000000)
                .reduce((acc, cur) => acc + cur ,0);
            const percentage = ( (conversion / total) *100);
            return <AccountsTable
                key={`${data.addr}${i}`}
                Rank={startNum++}
                Percentage={percentage.toFixed(2)}
                Addr={data.Addr}
                Balance={conversion}
                TxCountAccount={data.TxCountAccount}
                detailAccountHandler={this.props.detailAccountHandler}
            />
        });

        let combinedClasses = ['responsive-table', classes.table];
        return (
            <table className={combinedClasses.join(' ')}>
                <thead className={classes.tHead}>
                <tr>
                    <th scope="col">Rank</th>
                    <th scope="col">Address</th>
                    <th scope="col">Balance</th>
                    <th scope="col">Percentage</th>
                    <th scope="col">TxCount</th>
                </tr>
                </thead>
                {table}
            </table>
        );
    }
}
export default AccountTable;
