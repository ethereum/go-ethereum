// isNullOrUndefined returns true if a variable is null or undefined.
export const isNullOrUndefined = variable => variable === null || typeof variable === 'undefined';

export const mapChildren = (children, mapFunc) => !Array.isArray(children) || children.length < 1 ||
children.length === 1 ? mapFunc(children) : children.map(mapFunc);

export const Clearfix = () => <div className="clearfix"/>;

export const MEMORY_SAMPLE_LIMIT = 200; // Maximum number of memory data samples.
export const TRAFFIC_SAMPLE_LIMIT = 200; // Maximum number of traffic data samples.