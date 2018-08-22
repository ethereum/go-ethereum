import React, { Component } from 'react';
import InternalTable from './internalTable';
import Loading from '../../UI materials/loading'
import classes from './table.css';
import axios from "axios/index";

class InternalTransactionsTable extends Component {
    constructor(props) {
        super(props);
        this.state = {
            data: []
        };
    }

    async componentDidMount() {
        try {
            const response = await axios.get("http://localhost:8080/api/get_internal_transactions/");
            await this.setState({data: response.data});
        } catch (err) {
            console.log(err);
        }
    }

    render() {
        const table = this.state.data.map((data, i) => {  
            const conversion = data.Rewards / 10000000000000000000;
            return <InternalTable
                key={`${data.TxHash}${i}`}
                Hash={data.Hash}
                Action={data.Action}
                To={data.To}
                From= {data.From}
                Gas={data.Gas}
                GasUsed={data.GasUsed}
                ID={data.ID}
                Input={data.Input}
                Output={data.Output}
                Time={data.Time}
                Value={data.Value}    
                detailInternalHandler={this.props.detailInternalHandler}            
            />
        });

        let combinedClasses = ['responsive-table', classes.table];
        return (
            <div>
                <table className={combinedClasses.join(' ')}>
                    <thead>
                    <tr>                    
                        <th scope="col" className={classes.thItem}> Block Hash </th>
                        <th scope="col" className={classes.thItem}> Action </th>
                        <th scope="col" className={classes.thItem}> To </th>
                        <th scope="col" className={classes.thItem}> From </th>
                        <th scope="col" className={classes.thItem}> Gas </th>
                        <th scope="col" className={classes.thItem}> Gas Used</th>
                        <th scope="col" className={classes.thItem}> ID </th>
                        <th scope="col" className={classes.thItem}> Input </th>
                        <th scope="col" className={classes.thItem}> Output </th>
                        <th scope="col" className={classes.thItem}> Time </th>
                        <th scope="col" className={classes.thItem}> Value </th>
                    </tr>
                    </thead>
                    {table}
                </table>
            </div>
        );
    }
}
export default InternalTransactionsTable;
