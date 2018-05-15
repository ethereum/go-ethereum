import React from 'react';
import classes from '../nav.css';

const blocksDetailHeader = (props) => {
    return (
        <div className={classes.Secondary}>
            <span className={classes.Transactions}>Block# {props.blockNumber}</span>
        </div>
    )
}

export default blocksDetailHeader;