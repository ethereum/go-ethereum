import React from 'react';
import classes from './table.css';
import { Link } from 'react-router-dom'

const InternalTable = (props) => {
    return (
        <tbody>
            <tr style={{ borderTop: '1px solid #e0defb' }}>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} ><Link to="/blocks" style={{ color: '#8f67c9' }} onClick={() => props.detailBlockHandler(props.Number)}>
                    {props.Hash}
                </Link></td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'   }} >{props.Action}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'   }}>{props.To}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.From}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'   }}>{props.Gas}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} className={classes.GasUsed}><Link to="/mined/blocks" style={{ color: '#8f67c9' }} onClick={() => props.getBlocksMined(props.Coinbase)}>{props.Coinbase}</Link></td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.ID}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.Input}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'   }}>{props.Output}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.Time}</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.Value}</td>
            </tr>
            </tbody>
    )
}

export default InternalTable;
