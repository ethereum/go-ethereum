/// <reference types="node" />
export interface FormDataEntry {
    key: string;
    value: string | Blob | Buffer;
    fileName?: string;
}
export declare class FormData {
    private _entries;
    append(key: string, value: string | Blob | Buffer, fileName?: string): void;
}
export declare function getFormDataEntries(fd: FormData): FormDataEntry[];
