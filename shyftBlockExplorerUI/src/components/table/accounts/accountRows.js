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
        const table = this.state.data.map((data, i) => {
            return <AccountsTable
                key={`${data.addr}${i}`}
                Addr={data.Addr}
                Balance={data.Balance}
                TxCountAccount={data.TxCountAccount}
                detailAccountHandler={this.props.detailAccountHandler}
            />
        })

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
