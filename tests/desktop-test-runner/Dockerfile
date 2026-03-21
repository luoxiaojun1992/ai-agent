FROM mcr.microsoft.com/playwright:v1.58.2-noble

WORKDIR /app

COPY tests/desktop-test-runner/package.json tests/desktop-test-runner/package-lock.json ./
RUN npm ci

WORKDIR /app/desktop-client
COPY desktop-client/package.json desktop-client/package-lock.json ./
RUN npm ci

WORKDIR /app
COPY tests/desktop-test-runner/playwright.config.js ./
COPY tests/desktop-test-runner/tests ./tests
COPY desktop-client ./desktop-client

RUN mkdir -p /app/allure-results /app/test-results /app/artifacts

CMD ["xvfb-run", "-a", "npm", "test"]
