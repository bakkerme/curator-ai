# Curator Backend

Go-based pipeline engine and API server for the Curator platform.

## Architecture

- **API Layer** (`/api`): HTTP API using Gin framework
- **Pipeline** (`/pipeline`): Content processing pipeline engine
- **Adapters** (`/adapters`): Source connectors for Reddit, RSS, etc.
- **LLM** (`/llm`): LLM service abstraction layer
- **Storage** (`/storage`): BadgerDB persistence layer
- **Config** (`/config`): Configuration management

## Development

```bash
# Install dependencies
go mod tidy

# Run the server
go run cmd/curator/main.go

# Build for production
go build -o bin/curator ./cmd/curator
```

## Configuration

The backend can be configured via:
1. Environment variables
2. YAML configuration file (`./configs/curator.yaml`)
3. Default values

### Environment Variables

- `CURATOR_CONFIG`: Path to configuration file
- `PORT`: Server port (overrides config)

### Configuration File Example

```yaml
server:
  port: "8080"
  host: "0.0.0.0"

database:
  path: "./data/badger"

llm:
  provider: "ollama"
  endpoint: "http://localhost:11434"
  model: "llama3.2:3b"

pipeline:
  config_path: "./configs/pipeline.yaml"
  data_path: "./data/pipeline"
```

## API Endpoints

### Health & Status
- `GET /api/v1/health` - Health check
- `GET /api/v1/status` - Service status and configuration

### Pipeline Management
- `GET /api/v1/pipeline/config` - Get pipeline configuration
- `POST /api/v1/pipeline/config` - Update pipeline configuration
- `GET /api/v1/pipeline/status` - Get pipeline execution status
- `POST /api/v1/pipeline/run` - Trigger pipeline execution

## Next Steps

1. Implement pipeline configuration system
2. Add source adapters (Reddit, RSS)
3. Integrate LLM service layer
4. Build content processing pipeline
5. Add WebSocket support for real-time updates