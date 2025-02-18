# Use an official Node.js runtime as a parent image
FROM node:16

# Set the working directory inside the container
WORKDIR /usr/src/app

# Install dependencies
COPY package.json package-lock.json ./
RUN npm install

# Copy the Hardhat project files into the container
COPY . .

# Expose the port (optional depending on your app)
EXPOSE 8545

# Default command to run when the container starts
CMD ["npm", "run", "start"]
