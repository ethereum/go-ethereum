import React from 'react';
import classes from './table.css';
import { Link } from 'react-router-dom'

const BlockTable = (props) => {
    return (
          <tbody>
            <tr>
                <td style={{paddingLeft: '30pt' }} ><Link to="/blocks/detail" onClick={() => props.detailBlockHandler(props.Number)}>
                    {props.Number}
                </Link></td>
                <td style={{paddingLeft: '30pt' }} className={classes.addressTag}>{props.Hash}</td>
                <td style={{paddingLeft: '30pt' }}>{props.AgeGet}</td>
                <td style={{paddingLeft: '30pt' }}>{props.TxCount}</td>
                <td style={{paddingLeft: '30pt' }}>{props.UncleCount}</td>
                <td style={{paddingLeft: '30pt' }} className={classes.addressTag}><Link to="/mined/blocks" onClick={() => props.getBlocksMined(props.Coinbase)}>{props.Coinbase}</Link></td>
                <td style={{paddingLeft: '30pt' }}>{props.GasUsed}</td>
                <td style={{paddingLeft: '30pt' }}>{props.GasLimit}</td>
                <td style={{paddingLeft: '30pt' }}>TBD</td>
                <td style={{paddingLeft: '30pt' }}>{props.Reward}</td>
            </tr>
            </tbody>
    )
}

export default BlockTable;
