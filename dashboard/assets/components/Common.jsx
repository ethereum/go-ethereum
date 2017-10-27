// isNullOrUndefined returns true if the given variable is null or undefined.
export const isNullOrUndefined = variable => variable === null || typeof variable === 'undefined';

// defaultZero returns 0 if the given element is null or undefined, otherwise returns the given element.
export const defaultZero = elem => isNullOrUndefined(elem) ? 0 : elem;

export const MEMORY_SAMPLE_LIMIT = 200; // Maximum number of memory data samples.
export const TRAFFIC_SAMPLE_LIMIT = 200; // Maximum number of traffic data samples.

// The sidebar menu and the main content are rendered based on these elements.
export const TAGS = (() => {
    const T = {
        home: { title: "Home", },
        logs: { title: "Logs", },
        networking: { title: "Networking", },
        txpool: { title: "Txpool", },
        blockchain: { title: "Blockchain", },
        system: { title: "System", },
    };
    // Using the key is circumstantial in some cases, so it is better to insert it also as a value.
    // This way the mistyping is prevented.
    for(let key in T) {
        T[key]['id'] = key;
    }
    return T;
})();

// Temporary - taken from Material-UI
export const DRAWER_WIDTH = 240;