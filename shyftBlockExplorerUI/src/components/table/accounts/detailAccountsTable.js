import React from 'react';
import classes from './table.css';
import { Link } from 'react-router-dom'

const DetailAccountsTable = (props) => {    
    let flag;
        if(props.addr === props.to) {
            flag = true
        }else {
            flag = false
        }
    let genFlag;
    if(props.txHash.indexOf("GENESIS") !== -1) {
        genFlag = false
    }else {
        genFlag = true
    }
    return (
          <tbody>
            <tr>
                <td className={classes.addressTag}>
                    <Link to="/transaction/details" className={genFlag ? "" : classes.disabled} onClick={() => props.detailTransactionHandler(props.txHash)}>
                        {props.txHash}</Link>
                </td>
                <td>{props.blockNumber}</td>
                <td>{props.age}</td>
                <td className={classes.fromTag}>{props.from}</td>
                <td><div className={ flag ? classes.incoming : classes.out }>{ flag ? "IN" : "OUT" }</div></td>
                <td>{props.to}</td>
                <td>{props.value}</td>
                <td>{props.cost}</td>
            </tr>
            </tbody>
    )
};

export default DetailAccountsTable;
