import React from 'react';
import classes from './table.css';
import { Link } from 'react-router-dom'

const TransactionTable = (props) => {
    console.log("in transaction table")
    console.log(props)
    window.bar = props
    let flag;
    if(props.txHash.indexOf("GENESIS") !== -1) {
        flag = false
    }else {
        flag = true
    }

    return (
        <tbody>
            <tr className={classes.border}>
                <td className={classes.tdItem}>
                    <Link to="/transactions" style={{ color: '#8f67c9' }} className={flag ? "" : classes.disabled} onClick={() => props.detailTransactionHandler(props.txHash)}>
                        {props.txHash}
                    </Link>
                </td>
                <td className={classes.tdItem}>
                    <Link  style={{ color: '#8f67c9' }}  to="/block/transactions" onClick={() => props.getBlockTransactions(props.blockNumber)}>
                    {props.blockNumber}
                    </Link>
                </td>
                <td className={classes.tdItem}>{props.age}</td>
                <td className={[flag ? classes.fromTag : classes.disabled, classes.tdItem ]}>
                    <Link  style={{ color: '#8f67c9' }}  to="/account/detail" onClick={() => props.detailAccountHandler(props.from)}>
                        {props.from}
                    </Link>
                </td>             
                <td className={classes.tdItem}>
                    <Link style={{ color: '#8f67c9' }} to="/account/detail" onClick={() => props.detailAccountHandler(props.to)}>
                        {props.to}
                    </Link>
                </td>
                <td className={classes.tdItem}> {props.value} </td>
                <td className={classes.tdItem}> {props.cost} </td>
            </tr>
        </tbody>
    )
}

export default TransactionTable;
