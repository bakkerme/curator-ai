# Curator - Personal Intelligence Platform

Curator is a self-hosted personal intelligence platform that transforms scattered information into structured intelligence for thought leaders and emerging influencers.

## Architecture

This is a monorepo containing:

- **Backend** (`/backend`): Go-based pipeline engine and API
- **Frontend** (`/frontend`): React-based management interface

## Quick Start

```bash
# Start all services (backend + frontend + ollama)
docker-compose up --build

# Development mode (both backend and frontend)
./scripts/dev/start.sh

# Or start services separately:
make docker-backend   # Backend API + Ollama only
make docker-frontend  # Frontend only
```

## Service URLs

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080/api/v1

## Project Structure

```
├── backend/           # Go pipeline engine
│   ├── adapters/      # Content source connectors
│   ├── pipeline/      # Processing pipeline core
│   ├── llm/          # LLM integration layer
│   ├── storage/      # BadgerDB and data persistence
│   ├── api/          # HTTP API for frontend
│   └── config/       # Configuration management
├── frontend/         # React management interface
│   ├── src/          # TypeScript source code
│   └── public/       # Static assets
├── shared/           # Shared resources
│   ├── types/        # Generated TypeScript types
│   ├── schemas/      # JSON schemas
│   └── config/       # Shared configuration
├── cmd/              # Go application entry points
├── scripts/          # Development and deployment scripts
└── deployments/      # Docker and deployment configs
```

## Development

See individual component READMEs:
- [Backend Development](./backend/README.md)
- [Frontend Development](./frontend/README.md)

## Documentation

- [Product Vision](./planning/curator_pd_v2.md)
- [Technical Architecture](./planning/curator_tech_doc.md)
- [API Documentation](./docs/api/)