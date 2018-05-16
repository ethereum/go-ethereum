import React from 'react';
import classes from '../nav.css';

const transactionHeader = (props) => {
    return (
        <div className={classes.Secondary}>
        <span className={classes.Transactions}>Transactions</span>
      </div>
    )
}

export default transactionHeader;