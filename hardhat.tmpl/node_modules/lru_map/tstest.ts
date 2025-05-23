import {LRUMap} from './lru'

let m = new LRUMap<string, number>(3);
let entit = m.entries();
let k : string = entit.next().value[0];
let v : number = entit.next().value[1];
