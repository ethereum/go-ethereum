import { Transaction, DataFormat, DEFAULT_RETURN_FORMAT, FormatType } from 'web3-types';
import { ValidationSchemaInput } from 'web3-validator';
import { type CustomTransactionSchema } from '../types.js';
export declare function formatTransaction<ReturnFormat extends DataFormat = typeof DEFAULT_RETURN_FORMAT, TransactionType extends Transaction = Transaction>(transaction: TransactionType, returnFormat?: ReturnFormat, options?: {
    transactionSchema?: ValidationSchemaInput | CustomTransactionSchema | undefined;
    fillInputAndData?: boolean;
}): FormatType<TransactionType, ReturnFormat>;
//# sourceMappingURL=format_transaction.d.ts.map