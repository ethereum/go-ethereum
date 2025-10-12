// Fixed typo in variable name
const processData = (data) => {
  // Fixed: was 'procesedData'
  const processedData = data.map(item => ({
    id: item.id,
    name: item.name,
    // Fixed: was 'desciption'
    description: item.description
  }));
  
  return processedData;
};

// Fixed import statement
import { validateInput } from './validation';
// Fixed: was 'import { validateInput } from './validaton';'

export { processData };
