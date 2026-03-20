FROM node:18-alpine

WORKDIR /app
COPY package.json ./
RUN npm install
COPY api.test.js ./

CMD ["npm", "test"]
