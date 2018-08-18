import React from 'react';
import classes from './accounts.css';
import { Link } from 'react-router-dom'

const AccountsTable = (props) => {
    return (
        <tbody>
            <tr style={{ borderTop: '1px solid #e0defb' }}>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} >{props.Rank}</td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }}  className={classes.addressTag}><Link style={{ color: '#8f67c9' }} to="/account/detail" onClick={() => props.detailAccountHandler(props.Addr)}>
                    {props.Addr}
                </Link></td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} >{props.Balance}</td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} >{props.Percentage}%</td>
                <td style={{ paddingLeft: '30pt', paddingBottom: '7.5pt', paddingTop: '7.5pt'  }} >{props.AccountNonce}</td>
            </tr>
        </tbody>
    )
};

export default AccountsTable;
