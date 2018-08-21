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
            <tr style={{ borderTop: '1px solid #e0defb' }}>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} className={classes.addressTag}>
                    <Link to="/block/transactions" className={genFlag ? "" : classes.disabled} onClick={() => props.detailTransactionHandler(props.txHash)}>
                        {props.txHash}</Link>
                </td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.blockNumber}</td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.age}</td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} className={classes.fromTag}>{props.from}</td>
                <td><div className={ flag ? classes.incoming : classes.out }>{ flag ? "IN" : "OUT" }</div></td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.to}</td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.value}</td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.cost}</td>
            </tr>
            </tbody>
    )
};

export default DetailAccountsTable;
