# Go Product Architecture

This document describes the technical architecture of the Go Product system, focusing on the major components, their interactions, and design principles.

## System Overview

Go Product is a monolithic application built with Go 1.24 that implements an agent-based system with messaging capabilities. The architecture follows clean separation of concerns through multiple internal packages that handle different aspects of the system.

![Architecture Diagram](https://mermaid.ink/img/pako:eNp1ksFqwzAMhl_F-NRCHkDQYYcOVrKLYYcQfLOcLp0TWo9iF_aUvftke16brTBsMPrQ_31IMZJ72ZqEnMpgNFUBU-QvBL8iJbxrSUbSYfLsKGRNMcwG_qYfaPxl5jM1i1OB_5BVDXkW4xJ20KOiIUqSKwQVs3UZbWOMBQpKqQNbiBYzIhGnyCG1YB7N4_DQBT-0sJaLz-C9scYFPwRvpNjbk2N-mSiKmTOaOEcDprBkycDx3A1dn4JhlxRjMwALXW9C1aFXBQeKVlPGpPQdnJ79jrPaRbKzAQcnLLXnMnrWyJX0hkrskHU3K61bDfPZiHVGvnPLgX0KzaE23a-e23ULO-qQmGQkk8-GUmRG-6dGluPR_avpW1pqdXfg9fFndbMFOTSlYkL3UqitQenJkdQbSR-v0qPUQqmpTqpLKXSpb_UXr5qipw?type=png)

## Core Components

### 1. Runtime Context

The `RuntimeContext` serves as a central access point for system-wide resources:

- Provides thread-safe access to shared components
- Manages message bus and memory store instances
- Simplifies dependency injection throughout the system

### 2. Messaging System

The messaging system enables communication between entities within the application:

#### Components:
- **Message**: Core data structure representing communications
  - Content with MIME type support (text, JSON, commands)
  - Headers for metadata
  - Addressing (sender/recipients)
  - Unique ID and timestamps
  
- **MessageBus Interface**:
  - Abstracts message routing and delivery
  - Supports publish/subscribe pattern
  - Provides addressing mechanisms
  
- **MemoryMessageBus Implementation**:
  - Thread-safe in-memory implementation
  - Efficient routing algorithms
  - Support for direct, group, and broadcast messaging
  - Built-in subscription management

### 3. Entity System

Entities represent the actors in the system with different capabilities:

#### Entity Types:
- **ProductAgentEntity**: AI-powered agent that can process requests and generate responses
- **CliHumanEntity**: Represents a human user interacting through the command line
- **Group**: Collection of entities that can receive messages as a unit

#### Capabilities:
- Message sending/receiving
- Role-based permissions
- Metadata storage
- Lifecycle management (creation, activation, deactivation)

### 4. Tracing System

The tracing system provides comprehensive observability:

#### Components:
- **Tracer Interface**: Common API for all tracer implementations
- **EnhancedTracer**: High-level convenience methods (Info, Debug, Error)
- **FileTracer**: Buffered file-based implementation
- **ConsoleTracer**: Standard output logging
- **NoopTracer**: No-operation tracer for testing/production

#### Features:
- Configurable buffer sizes and flush intervals
- Multiple trace levels (Error, Warning, Info, Debug, Verbose)
- Component and operation tracking
- Thread-safe implementation

### 5. Memory System

The memory system provides persistent storage capabilities:

#### Components:
- **MemoryStore Interface**: Common API for all storage implementations
- **FileMemoryStore**: JSON file-based implementation
- **MemoryRecord**: Core data structure for memory entries

#### Features:
- Rich metadata
- Content type support
- Importance levels
- Expiration handling
- Soft delete capabilities

### 6. Chat Interface

The chat interface provides human interaction with the system:

#### Components:
- **Chat**: Basic chat implementation
- **EnhancedChat**: Advanced chat with message bus integration
- **Command**: Special chat commands for system control

## Communication Flow

1. **User Input**: Human enters text through CLI
2. **Message Creation**: Input is converted to a Message by CliHumanEntity
3. **Message Bus**: Routes message to appropriate recipient(s)
4. **Agent Processing**: ProductAgentEntity receives message and processes it
5. **LLM Generation**: Agent uses language model to generate a response
6. **Return Flow**: Response follows reverse path to user
7. **Tracing**: All operations are logged through the tracing system

## Design Principles

### 1. Interface-Based Design
- All major components expose interfaces
- Enables multiple implementations and testing
- Clear contracts between components

### 2. Thread Safety
- All shared resources are protected by mutexes
- Concurrent access is safely managed
- Channel-based communication where appropriate

### 3. Clean Package Structure
- Each package has a clear responsibility
- Minimized dependencies between packages
- No circular dependencies

### 4. Extensibility
- Easy to add new entity types
- Support for multiple message bus implementations
- Pluggable tracing system

### 5. Error Handling
- Consistent error propagation
- Proper resource cleanup with defer
- Graceful degradation

## Configuration

The system is configured at startup in `cmd/myapp/main.go`:

1. Initialize tracing system
2. Create message bus
3. Set up runtime context
4. Initialize LLM
5. Create memory store
6. Create and configure entities
7. Start enhanced chat interface

## Future Considerations

1. **Distributed Message Bus**: Replace in-memory implementation with Redis/Kafka
2. **Web Interface**: Add HTTP API and web-based UI
3. **Plugin System**: Allow dynamic loading of entity implementations
4. **Authentication**: Add user authentication and authorization
5. **Rate Limiting**: Implement throttling for message processing

## Reference Documentation

For detailed package documentation, see the inline comments and package docs within the codebase. For usage instructions, see the [README.md](README.md).
