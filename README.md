# Go Product

A monolithic Go application with agent-based messaging capabilities, structured to support multiple local packages.

## Project Structure

```
/
├── cmd/
│   └── myapp/       # Application entry point
├── internal/        # Private packages
│   ├── hello/       # Hello world functionality
│   ├── entity/      # Entity system (humans, agents, groups)
│   ├── memory/      # Memory storage system
│   ├── messaging/   # Messaging system
│   └── common/      # Shared utilities
├── bin/             # Build output
├── Makefile         # Build automation
└── Dockerfile       # For cross-compilation
```

## Requirements

- Go 1.24 or higher
- Docker (for cross-compilation)

## Setup Instructions

1. Clone the repository
2. Run `make deps` to install dependencies
3. Run `make build` to build for your current platform

## Usage Instructions

- `make help` - Show available make targets
- `make check` - Check for required dependencies
- `make deps` - Install dependencies
- `make build` - Build for current platform
- `make run` - Build and run the application
- `make clean` - Clean up build artifacts
- `make test` - Run tests
- `make fmt` - Format Go code with standard formatter
- `make format` - Alias for `make fmt`
- `make docker` - Build cross-platform binaries (Windows x64, Linux x64, macOS arm64)

### Tracing

The application generates trace logs in `./trace.log` by default. Tracing can be monitored in real-time with:

```bash
tail -f ./trace.log
```

## Core Components

### Messaging System

The messaging system provides a flexible infrastructure for entity communication:

- **MessageBus Interface**: Routes messages between entities
- **In-Memory Implementation**: Thread-safe message routing
- **Addressing**: UUID-based addressing for entities
- **Message Types**: Supports text, JSON, and command messages
- **Messaging Patterns**:
  - Direct messaging (entity-to-entity)
  - Group messaging (entity-to-group)
  - Broadcast messaging (entity-to-all)
- **Tracing Support**: Integrated message tracing for debugging and monitoring
- **Runtime Integration**: Available via the `RuntimeContext` for system-wide access

### Entity System

Provides a framework for different types of actors in the system:

- Human entities
- Agent entities
- System entities
- Group entities

### Tracing System

Comprehensive tracing infrastructure for monitoring system operations:

- **Multiple Output Options**:
  - File-based tracing with configurable buffering
  - Console output for debugging
  - No-op tracer for production environments
- **Buffered File Tracing**: 
  - Default 4KB buffer size or time-based (5s) flushing
  - Configurable buffer size and flush intervals
  - Append or overwrite options
- **Trace Events**:
  - Component-specific events (messaging, memory, entity, etc.)
  - Operation tracking (send, receive, create, delete, etc.)
  - Metadata and context capture
- **Thread Safety**: All tracers are thread-safe for concurrent access

## Output Locations

- Default build: `./bin/myapp`
- Cross-platform builds:
  - Windows AMD64: `./bin/windows/amd64/myapp.exe`
  - Linux AMD64: `./bin/linux/amd64/myapp`
  - macOS ARM64: `./bin/macos/arm64/myapp`
