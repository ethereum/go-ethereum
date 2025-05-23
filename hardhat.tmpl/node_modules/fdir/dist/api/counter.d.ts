export declare class Counter {
    private _files;
    private _directories;
    set files(num: number);
    get files(): number;
    set directories(num: number);
    get directories(): number;
    /**
     * @deprecated use `directories` instead
     */
    get dirs(): number;
}
