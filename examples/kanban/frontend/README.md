# Kanban Frontend (React + Tailwind CSS v4)

A React-based frontend for the Kanban task management system, demonstrating SnapSQL's capabilities.

## Features

- **Board Management**: View and manage the active board
- **List Columns**: Display tasks organized by stage (Backlog, In Progress, Review, Done)
- **Card Operations**: Create, update, move cards between lists
- **Card Details**: View and edit card details with comments
- **Iteration Completion**: Complete iterations and create new boards with task migration

## Prerequisites

- Node.js 18+ and npm/pnpm/yarn
- Backend API running on `http://localhost:8080` (or configure via `.env`)

## Setup

1. Copy the environment configuration:

```bash
cp .env.example .env
```

2. Install dependencies:

```bash
npm install
```

3. Start the development server:

```bash
npm run dev
```

The application will be available at `http://localhost:5173`.

## Configuration

Create a `.env` file based on `.env.example`:

```env
# For local development, leave empty to use the proxy
VITE_API_BASE_URL=

# For production or custom API endpoint
# VITE_API_BASE_URL=http://your-api-server:8080
```

The development server includes a proxy configuration that forwards all `/api` requests to `http://localhost:8080`. This eliminates CORS issues and makes the frontend and backend appear to be on the same origin during development.

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint
- `npm run test` - Run tests with Vitest

## Architecture

### Components

- `BoardView` - Main board container showing lists and cards
- `StageColumn` - Individual list column component
- `Card` - Card item display component
- `CardDetailDrawer` - Drawer for viewing/editing card details
- `Dialog` - Modal dialog components (create card, etc.)

### Hooks

- `useBoard` - Custom hook for board state management and API operations

### API Client

- `api/client.ts` - Base HTTP request wrapper
- `api/kanban.ts` - Kanban-specific API methods
- `api/types.ts` - TypeScript type definitions

## Technology Stack

- **React 19** - UI framework
- **TypeScript** - Type safety
- **Tailwind CSS v4** - Styling
- **Vite** - Build tool
- **Vitest** - Testing framework

## License

Same as the parent SnapSQL project.

// eslint.config.js
import reactX from 'eslint-plugin-react-x'
import reactDom from 'eslint-plugin-react-dom'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      // Other configs...
      // Enable lint rules for React
      reactX.configs['recommended-typescript'],
      // Enable lint rules for React DOM
      reactDom.configs.recommended,
    ],
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```
