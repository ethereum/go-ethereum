import React from 'react';
import classes from '../nav.css';

const accountsDetailHeader = (props) => {
    return (
        <div className={classes.Secondary}>
            <span className={classes.Transactions}>Account# {props.addr}</span>
        </div>
    )
}

export default accountsDetailHeader;