import React from 'react';
import classes from '../nav.css';

const transactionDetailHeader = (props) => {
    return (
        <div className={classes.Secondary}>
            <span className={classes.Transactions}>Transaction</span>
        </div>
    )
}

export default transactionDetailHeader;