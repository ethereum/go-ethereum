import React from 'react';
import classes from './accounts.css';
import { Link } from 'react-router-dom'

const AccountsTable = (props) => {
    return (
          <tbody>
            <tr>
                <td>1</td>
                <td className={classes.addressTag}><Link to="/account/detail" onClick={() => props.detailAccountHandler(props.Addr)}>
                    {props.Addr}
                </Link></td>
                <td>{props.Balance}</td>
                <td>12.01%</td>
                <td>{props.TxCountAccount}</td>
            </tr>
            </tbody>
    )
}

export default AccountsTable;
