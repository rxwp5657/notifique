# Notifique Service
A notification system with support for multiple delivery channels and priority queues.

## Features

- Multiple delivery channels (email, in-app)
- Priority queues for message delivery
- Template-based notifications
- Distribution lists
- Rate limiting
- Response caching
- Live notifications via server-sent events.

## Architecture

The system supports different backend configurations:

- PostgreSQL + RabbitMQ
- PostgreSQL + SQS
- DynamoDB + RabbitMQ
- DynamoDB + SQS

## Getting Started

1. Clone the repository
2. Create a `.env` file with required configuration
3. Run dependencies (database, message queue, Redis)
4. Start the server:
```bash
go run cmd/main.go
```

## API Documentation

The API is documented using OpenAPI 3.0. See `api/openapi.yaml` for full specification.

## Development

The project uses:
- Go 1.21+
- Wire for dependency injection 
- Gin web framework
- Redis for caching/rate limiting
- PostgreSQL/DynamoDB for persistence
- RabbitMQ/SQS for message queues

## License

MIT