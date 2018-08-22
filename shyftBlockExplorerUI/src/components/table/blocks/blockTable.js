import React from 'react';
import classes from './table.css';
import { Link } from 'react-router-dom'

const BlockTable = (props) => {
    return (
        <tbody>
            <tr className={classes.border}>
                <td className={classes.tdItem}>
                    <Link to="/blocks" style={{ color: '#8f67c9' }} onClick={() => props.detailBlockHandler(props.Number)}>
                        {props.Number}
                    </Link>
                </td>
                <td className={classes.tdItem}> {props.Hash} </td>
                <td className={classes.tdItem}> {props.AgeGet} </td>
                <td className={classes.tdItem}> {props.TxCount} </td>
                <td className={classes.tdItem}> {props.UncleCount} </td>
                <td className={classes.tdItem}>
                    <Link to="/mined/blocks" style={{ color: '#8f67c9' }} onClick={() => props.getBlocksMined(props.Coinbase)}>
                        {props.Coinbase}
                    </Link>
                </td>
                <td className={classes.tdItem}> {props.GasUsed} </td>
                <td className={classes.tdItem}> {props.GasLimit} </td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'   }}>TBD</td>
                <td style={{paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}>{props.Reward}</td>
            </tr>
            </tbody>
    )
}

export default BlockTable;
