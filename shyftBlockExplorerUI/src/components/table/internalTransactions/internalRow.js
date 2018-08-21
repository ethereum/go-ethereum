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

        console.log("in table row component");

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
                        <th scope="col" className={classes.blockHash} style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e" }} >Block Hash</th>
                        <th scope="col" className={classes.action} style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e" }} >Action</th>
                        <th scope="col" className={classes.to} style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e" }} >To</th>
                        <th scope="col" className={classes.from} style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e" }} >From</th>
                        <th scope="col" className={classes.gas} style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e" }} >Gas</th>
                        <th scope="col" className={classes.gasUsed} style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e" }} >Gas Used</th>
                        <th scope="col" className={classes.id} style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e" }}>ID</th>
                        <th scope="col" className={classes.input} style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e" }}>Input</th>
                        <th scope="col" className={classes.output} style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e" }}>Output</th>
                        <th scope="col" className={classes.time} style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e" }}>Time</th>
                        <th scope="col" className={classes.value} style={{fontSize: "8pt", backgroundColor: "white", color: "#4f2e7e" }}>Value</th>
                    </tr>
                    </thead>
                    {table}
                </table>
            </div>
        );
    }
}
export default InternalTransactionsTable;
