import React from 'react';
import classes from './table.css';
import arrow from '../../assets/arrow_right.png';
import { Link } from 'react-router-dom'

const TransactionTable = (props) => {
    let flag;
    if(props.txHash.indexOf("GENESIS") != -1) {
        flag = false
    }else {
        flag = true
    }

    return (
          <tbody>
            <tr>
                <td className={classes.addressTag}>
                    <Link to="/transaction/details" className={flag ? "" : classes.disabled} onClick={() => props.detailTransactionHandler(props.txHash)}>
                        {props.txHash}</Link>
                </td>
                <td>{props.blockNumber}</td>
                <td>{props.age}</td>
                <td className={classes.fromTag}>{props.from}</td>
                <td><img className={classes.arrow} src={arrow} alt="arrow"/></td>
                <td className={classes.toTag}>{props.to}</td>
                <td>{props.value}</td>
                <td>{props.cost}</td>
            </tr>
            </tbody>
    )
}

export default TransactionTable;
