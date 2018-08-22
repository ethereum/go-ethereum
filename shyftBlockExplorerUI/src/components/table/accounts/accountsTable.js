import React from 'react';
import classes from './table.css';
import { Link } from 'react-router-dom'

const AccountsTable = (props) => {
    return (
        <tbody>
            <tr className={classes.border}>
                <td className={classes.tdItem}>{props.Rank}</td>
                <td className={classes.tdItem}>
                    <Link className={classes.tdLink} to="/account/detail" onClick={() => props.detailAccountHandler(props.Addr)}>
                    {props.Addr}
                    </Link>
                </td>
                <td className={classes.tdItem}> {props.Balance} </td>
                <td className={classes.tdItem}> {props.Percentage}% </td>
                <td className={classes.tdItem}> {props.AccountNonce} </td>
            </tr>
        </tbody>
    )
};

export default AccountsTable;
