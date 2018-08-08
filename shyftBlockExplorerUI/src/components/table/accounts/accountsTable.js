import React from 'react';
import classes from './accounts.css';
import { Link } from 'react-router-dom'

const AccountsTable = (props) => {
    return (
          <tbody>
            <tr>
                <td>{props.Rank}</td>
                <td className={classes.addressTag}><Link to="/account/detail" onClick={() => props.detailAccountHandler(props.Addr)}>
                    {props.Addr}
                </Link></td>
                <td>{props.Balance}</td>
                <td>{props.Percentage}%</td>
                <td>{props.AccountNonce}</td>
            </tr>
            </tbody>
    )
};

export default AccountsTable;
