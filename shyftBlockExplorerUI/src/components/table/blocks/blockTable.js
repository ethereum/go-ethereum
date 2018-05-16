import React, { Component } from 'react';
import classes from './table.css';
import arrow from '../../assets/arrow_right.png';
import { Link } from 'react-router-dom'

const BlockTable = (props) => {
    return (
          <tbody key={props.key}>
            <tr>
                <td><Link to="/blocks/detail" onClick={() => props.detailBlockHandler(props.Number)}>
                    {props.Number}
                </Link></td>
                <td className={classes.addressTag}>{props.Hash}</td>
                <td>{props.Age}</td>
                <td>{props.TxCount}</td>
                <td>{props.UncleCount}</td>
                <td className={classes.addressTag}>{props.Coinbase}</td>
                <td>{props.GasUsed}</td>
                <td>{props.GasLimit}</td>
                <td>12.01</td>
                <td>3.2</td>
            </tr>
            </tbody>
    )
}

export default BlockTable;
