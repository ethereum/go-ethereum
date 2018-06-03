import React from 'react';
import classes from '../nav.css';

const accountsDetailHeader = (props) => {
    const total = props.data
        .map(num => Number(num.Amount) / 10000000000000000000)
        .reduce((acc, cur) => acc + cur ,0);
    return (
        <div className={classes.Secondary}>
            <span className={classes.Transactions}>Account# {props.addr}</span>
            <span className={classes.Transactions}>Total Value: {total}</span>
        </div>
    )
};

export default accountsDetailHeader;