import React, { Component } from 'react';
import classes from './table.css';
import arrow from '../../assets/arrow_right.png';

const BlockTable = (props) => {
    return (
          <tbody key={props.key}>
            <tr>
                <td>{props.Number}</td>
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
