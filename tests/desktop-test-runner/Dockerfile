FROM mcr.microsoft.com/playwright:v1.58.2-noble

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY playwright.config.js ./
COPY tests ./tests
COPY start-desktop-tests.sh ./start-desktop-tests.sh
RUN chmod +x /app/start-desktop-tests.sh

RUN mkdir -p /app/allure-results /app/test-results /app/artifacts

CMD ["/app/start-desktop-tests.sh"]
