import React from 'react';
import classes from '../nav.css';

const accountsHeader = (props) => {
    return (
        <div className={classes.Secondary}>
            <span className={classes.Transactions}>Accounts</span>
        </div>
    )
}

export default accountsHeader;