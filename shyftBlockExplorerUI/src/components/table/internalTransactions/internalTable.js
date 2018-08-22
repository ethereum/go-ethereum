import React from 'react';
import classes from './table.css';
import { Link } from 'react-router-dom'
import Button from 'react-bootstrap/lib/Button';

const InternalTable = (props) => {
    return (
        <tbody>
            <tr className={classes.border}>
                <td className={classes.tdItem}>
                    <div className={classes.tdLink} onClick={() => props.detailInternalHandler(props.Hash)}>
                        {props.Hash}
                    </div>
                </td>
                <td className={classes.tdItem}>{props.Action}</td>
                <td className={classes.tdItem}>{props.To}</td>
                <td className={classes.tdItem}>{props.From}</td>
                <td className={classes.tdItem}>{props.Gas}</td>
                <td className={classes.tdItem}> {props.GasUsed} </td>
                <td className={classes.tdItem}>{props.ID}</td>
                <td className={classes.tdItem}> 
                    <div className={classes.tdLink} onClick={()=>alert( props.Input )}> Show Input   
                    <input type="hidden" id={"input" + props.Hash} value={props.Input} /> </div> 
                </td>
                <td className={classes.tdItem}> 
                    <div className={classes.tdLink} onClick={()=>alert(props.Output)}> Show Output 
                    <input type="hidden" id={"output" + props.Hash} value={props.Output} /> </div> 
                </td>
                <td className={classes.tdItem}>{props.Time}</td>
                <td className={classes.tdItem}>{props.Value}</td>  
            </tr>
        </tbody>
    )
}

export default InternalTable;
