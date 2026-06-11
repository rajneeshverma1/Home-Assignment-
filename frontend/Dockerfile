FROM node:24-alpine

WORKDIR /app

# Install dependencies
COPY package.json package-lock.json ./
RUN npm ci

# Copy source files
COPY . .

# Build Next.js project
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# Expose web server port
EXPOSE 3000

# Start server
CMD ["npm", "start"]
