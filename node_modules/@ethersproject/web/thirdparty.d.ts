declare module "node-fetch" {
    function fetch(url: string, options: any): Promise<Response>;
    export default fetch;
}
