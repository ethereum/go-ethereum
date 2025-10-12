/**
 * Optimized algorithm implementation
 * Improved performance and memory usage
 */

// Optimized sorting algorithm
function optimizedSort(array, compareFn) {
  // Use native sort for better performance
  return array.slice().sort(compareFn);
}

// Optimized data structure
class OptimizedMap {
  constructor() {
    this.data = new Map();
    this.size = 0;
  }

  set(key, value) {
    const exists = this.data.has(key);
    this.data.set(key, value);
    
    if (!exists) {
      this.size++;
    }
    
    return this;
  }

  get(key) {
    return this.data.get(key);
  }

  has(key) {
    return this.data.has(key);
  }

  delete(key) {
    const exists = this.data.has(key);
    if (exists) {
      this.data.delete(key);
      this.size--;
      return true;
    }
    return false;
  }

  clear() {
    this.data.clear();
    this.size = 0;
  }
}

// Memory-efficient data processing
function processLargeDataset(dataset) {
  const results = [];
  const batchSize = 1000;
  
  for (let i = 0; i < dataset.length; i += batchSize) {
    const batch = dataset.slice(i, i + batchSize);
    const processed = batch.map(item => ({
      ...item,
      processed: true
    }));
    
    results.push(...processed);
    
    // Allow garbage collection
    if (i % 10000 === 0) {
      await new Promise(resolve => setImmediate(resolve));
    }
  }
  
  return results;
}

module.exports = {
  optimizedSort,
  OptimizedMap,
  processLargeDataset
};
