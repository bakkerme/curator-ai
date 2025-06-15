# Curator Frontend

React-based management interface for the Curator platform.

## Features

- **Real-time Status Monitoring**: Live system and pipeline status updates
- **Pipeline Configuration**: Visual interface for configuring content processing pipelines
- **Analytics Dashboard**: Performance metrics and insights
- **Responsive Design**: Works on desktop and mobile devices

## Technology Stack

- **React 18**: Modern React with hooks and functional components
- **TypeScript**: Type-safe development
- **Vite**: Fast development server and build tool
- **TailwindCSS**: Utility-first CSS framework
- **Tanstack Query**: Server state management and caching
- **Axios**: HTTP client for API communication
- **Lucide React**: Modern icon library

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Type checking
npm run type-check

# Linting
npm run lint
```

## Configuration

The frontend is configured to proxy API requests to the backend server during development. In production, the backend serves the built frontend files.

### Environment Variables

Create a `.env.local` file for local development:

```
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

## Project Structure

```
src/
├── components/        # Reusable UI components
├── pages/            # Page components
├── hooks/            # Custom React hooks
├── services/         # API client and external services
├── utils/            # Utility functions
├── App.tsx           # Main application component
├── main.tsx          # Application entry point
└── index.css         # Global styles and Tailwind imports
```

## API Integration

The frontend communicates with the Go backend via REST API:

- **Health Check**: `GET /api/v1/health`
- **System Status**: `GET /api/v1/status`
- **Pipeline Status**: `GET /api/v1/pipeline/status`
- **Pipeline Config**: `GET/POST /api/v1/pipeline/config`
- **Run Pipeline**: `POST /api/v1/pipeline/run`

## Shared Types

TypeScript types are shared between frontend and backend via the `/shared` directory. Types are generated from JSON schemas defined in the backend.

## Next Steps

1. Implement pipeline configuration interface
2. Add real-time WebSocket connection for live updates
3. Build analytics dashboard with charts and metrics
4. Add user authentication and authorization
5. Implement pipeline template system