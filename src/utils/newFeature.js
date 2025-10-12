/**
 * New utility function for data processing
 * Provides enhanced functionality for common operations
 */

class DataProcessor {
  constructor(options = {}) {
    this.options = {
      timeout: 5000,
      retries: 3,
      ...options
    };
  }

  /**
   * Process data with enhanced error handling
   * @param {Array} data - Input data array
   * @returns {Promise<Array>} Processed data
   */
  async process(data) {
    try {
      const results = [];
      
      for (const item of data) {
        const processed = await this.processItem(item);
        results.push(processed);
      }
      
      return results;
    } catch (error) {
      console.error('Processing error:', error);
      throw new Error(`Failed to process data: ${error.message}`);
    }
  }

  /**
   * Process individual item
   * @param {Object} item - Single data item
   * @returns {Promise<Object>} Processed item
   */
  async processItem(item) {
    // Enhanced processing logic
    return {
      ...item,
      processed: true,
      timestamp: Date.now()
    };
  }
}

module.exports = DataProcessor;
