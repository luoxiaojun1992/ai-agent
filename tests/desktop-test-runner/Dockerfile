FROM mcr.microsoft.com/playwright:v1.58.2-noble

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY playwright.config.js ./
COPY tests ./tests

RUN mkdir -p /app/allure-results /app/test-results /app/artifacts

CMD ["xvfb-run", "-a", "npm", "test"]
