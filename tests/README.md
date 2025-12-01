# AI Agent Frontend Tests

This directory contains automated tests for the AI Agent frontend UI using Playwright.

## Prerequisites

1. Node.js (version 14 or newer)
2. npm (usually comes with Node.js)

## Setup

Install the required dependencies:

```bash
npm install
```

Install Playwright browsers (required for running tests):

```bash
npx playwright install
```

## Running Tests

### Headless mode (default)

```bash
npm test
```

Or directly with Playwright:

```bash
npx playwright test
```

### With UI mode

To run tests with Playwright's UI mode:

```bash
npm run test:ui
```

### Specific test files

To run a specific test file:

```bash
npx playwright test specs/chat.spec.js
```

## Viewing Reports

After running tests, you can view the HTML report:

```bash
npm run test:report
```

Or directly:

```bash
npx playwright show-report
```

## Test Structure

- `playwright.config.js` - Playwright configuration
- `specs/` - Test specifications
- `specs/chat.spec.js` - Chat functionality tests

## Writing New Tests

1. Add new test files in the `specs/` directory
2. Follow the existing patterns in `chat.spec.js`
3. Use Playwright's [documentation](https://playwright.dev/) for API references