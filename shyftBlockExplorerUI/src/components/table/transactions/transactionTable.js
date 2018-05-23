import React from 'react';
import classes from './table.css';
import arrow from '../../assets/arrow_right.png';
import { Link } from 'react-router-dom'

const TransactionTable = (props) => {
    let flag;
    if(props.txHash.indexOf("GENESIS") !== -1) {
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
                <td><Link to="/block/transactions" onClick={() => props.getBlockTransactions(props.blockNumber)}>{props.blockNumber}</Link></td>
                <td>{props.age}</td>
                <td className={flag ? classes.fromTag : classes.disabled }><Link to="/account/detail" onClick={() => props.detailAccountHandler(props.from)}>{props.from}</Link></td>
                <td><img className={classes.arrow} src={arrow} alt="arrow"/></td>
                <td className={classes.toTag}><Link to="/account/detail" onClick={() => props.detailAccountHandler(props.to)}>{props.to}</Link></td>
                <td>{props.value}</td>
                <td>{props.cost}</td>
            </tr>
            </tbody>
    )
}

export default TransactionTable;
