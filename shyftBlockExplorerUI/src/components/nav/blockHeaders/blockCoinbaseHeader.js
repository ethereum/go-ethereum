import React from 'react';
import classes from '../nav.css';

const blocksCoinbaseHeader = (props) => {
    return (
        <div className={classes.Secondary}>
            <span className={classes.Transactions}>Coinbase: {props.coinbase}</span>
        </div>
    )
};

export default blocksCoinbaseHeader;