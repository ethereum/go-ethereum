import React from 'react';
import classes from './table.css';
import arrow from '../../assets/arrow_right_black.png';
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
            <tr style={{ borderTop: '1px solid #e0defb' }}>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} className={classes.addressTag}>
                    <Link to="/transaction/details" style={{ color: '#8f67c9' }} className={flag ? "" : classes.disabled} onClick={() => props.detailTransactionHandler(props.txHash)}>
                        {props.txHash}</Link>
                </td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}><Link  style={{ color: '#8f67c9' }}  to="/block/transactions" onClick={() => props.getBlockTransactions(props.blockNumber)}>{props.blockNumber}</Link></td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.age}</td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} className={flag ? classes.fromTag : classes.disabled }><Link  style={{ color: '#8f67c9' }}  to="/account/detail" onClick={() => props.detailAccountHandler(props.from)}>{props.from}</Link></td>             
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} className={classes.toTag}><Link style={{ color: '#8f67c9' }} to="/account/detail" onClick={() => props.detailAccountHandler(props.to)}>{props.to}</Link></td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.value}</td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.cost}</td>
            </tr>
        </tbody>
    )
}

export default TransactionTable;
