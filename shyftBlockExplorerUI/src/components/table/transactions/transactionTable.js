import React, { Component } from 'react';
import classes from './table.css';
import arrow from '../../assets/arrow_right.png';
import { Link } from 'react-router-dom'

const TransactionTable = (props) => {
    return (
          <tbody>
            <tr>
                <td className={classes.addressTag}>
                    <Link to="/transaction/details" onClick={() => props.detailTransactionHandler(props.txHash)}>
                        {props.txHash}</Link>
                </td>
                <td>{props.blockNumber}</td>
                <td>30 secs ago</td>
                <td className={classes.fromTag}>{props.from}</td>
                <img className={classes.arrow} src={arrow}/>
                <td>{props.to}</td>
                <td>{props.value}</td>
                <td>{props.cost}</td>
            </tr>
            </tbody>
    )
}

export default TransactionTable;
