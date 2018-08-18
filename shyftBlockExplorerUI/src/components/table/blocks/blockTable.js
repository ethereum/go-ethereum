import React from 'react';
import classes from './table.css';
import { Link } from 'react-router-dom'

const BlockTable = (props) => {
    return (
          <tbody>
            <tr style={{ borderTop: '1px solid #e0defb' }}>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} ><Link to="/blocks/detail" style={{ color: '#8f67c9' }} onClick={() => props.detailBlockHandler(props.Number)}>
                    {props.Number}
                </Link></td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'   }} className={classes.addressTag}>{props.Hash}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'   }}>{props.AgeGet}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.TxCount}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'   }}>{props.UncleCount}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} className={classes.addressTag}><Link to="/mined/blocks" style={{ color: '#8f67c9' }} onClick={() => props.getBlocksMined(props.Coinbase)}>{props.Coinbase}</Link></td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.GasUsed}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.GasLimit}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'   }}>TBD</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.Reward}</td>
            </tr>
            </tbody>
    )
}

export default BlockTable;
