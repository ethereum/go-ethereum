import React from 'react';
import classes from './nav.css';

const blocksHeader = (props) => {
    return (
        <div className={classes.Secondary}>
            <span className={classes.Transactions}>Blocks</span>
        </div>
    )
}

export default blocksHeader;