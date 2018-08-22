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
            <tr className={classes.border}>
                <td className={classes.tdItem}>
                    <div className={[genFlag ? "" : classes.disabled, classes.tdLink]} onClick={() => props.detailTransactionHandler(props.txHash)}>
                        {props.txHash}
                    </div>
                </td>
                <td className={classes.tdItem}> {props.blockNumber} </td>
                <td className={classes.tdItem}> {props.age} </td>
                <td className={classes.tdItem}> {props.from} </td>
                <td>
                    <div className={ [flag ? classes.incoming : classes.out, classes.tdItem ]}>
                        { flag ? "IN" : "OUT" }
                    </div>
                </td>
                <td className={classes.tdItem}> {props.to} </td>
                <td className={classes.tdItem}> {props.value} </td>
                <td className={classes.tdItem}> {props.cost} </td>
            </tr>
        </tbody>
    )
};

export default DetailAccountsTable;
