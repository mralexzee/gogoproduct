# Developer Guide

This guide provides detailed information for developers working on the Go Product codebase.

## Getting Started

### Development Environment Setup

1. Install Go 1.24 or higher
2. Clone the repository
3. Run `make deps` to install dependencies
4. Run `make test` to ensure everything is working

### Development Workflow

1. Make your changes
2. Run `make fmt` to format code
3. Run `make test` to run unit tests
4. Run `make build` to build the application
5. Run `make run` to test your changes

## Working with the Messaging System

### Creating a Custom Message

```go
// Create a text message
textMsg := messaging.NewTextMessage(
    senderID,         // string
    []string{recipientID}, // []string
    "Hello, world!",       // string content
)

// Create a JSON message
jsonMsg := messaging.NewJSONMessage(
    senderID,         // string
    []string{recipientID}, // []string
    map[string]interface{}{
        "key": "value",
    },
)

// Create a command message
cmdMsg := messaging.NewCommandMessage(
    senderID,         // string
    []string{recipientID}, // []string
    "command_name",         // string
    map[string]interface{}{
        "param": "value",
    },
)
```

### Implementing a Custom Entity

To create a custom entity that works with the messaging system:

1. Implement the `entity.Entity` interface:

```go
type MyCustomEntity struct {
    id        string
    name      string
    messageBus messaging.MessageBus
    // other fields...
}

// Implement all required methods from the Entity interface
func (e *MyCustomEntity) ID() string {
    return e.id
}

// ... implement other required methods
```

2. Add message handling:

```go
func (e *MyCustomEntity) Start(ctx context.Context) error {
    // Subscribe to messages
    return e.messageBus.Subscribe(e.id, func(msg messaging.Message) error {
        // Process messages
        return nil
    })
}
```

## Working with the Tracing System

### Creating a Tracer

```go
// Create a file tracer
tracer, err := tracing.CreateFileTracer(
    "./logs/my_component.log", // file path
    5*time.Second,             // flush interval
    4096,                      // buffer size
)
if err != nil {
    // handle error
}
defer tracer.Close()
```

### Adding Trace Points

```go
// Log info message
tracer.Info("Component initialized with configuration: %v", config)

// Log debug information
tracer.Debug("Processing message: %s", messageID)

// Log errors
tracer.Error("Failed to connect to service: %v", err)

// Log warnings
tracer.Warning("Approaching resource limit: %d/%d", current, max)
```

### Creating a Custom Tracer

To implement a custom tracer:

```go
type MyCustomTracer struct {
    // fields
}

// Implement the Tracer interface
func (t *MyCustomTracer) Trace(event tracing.Event) error {
    // Custom implementation
    return nil
}

func (t *MyCustomTracer) Flush() error {
    // Custom implementation
    return nil
}

func (t *MyCustomTracer) Close() error {
    // Custom implementation
    return nil
}

func (t *MyCustomTracer) SetLevel(level tracing.Level) {
    // Custom implementation
}
```

## Common Pitfalls and Best Practices

### Thread Safety

All shared resources must be protected by mutexes:

```go
type ThreadSafeComponent struct {
    data map[string]interface{}
    mu   sync.RWMutex
}

func (c *ThreadSafeComponent) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.data[key]
    return val, ok
}

func (c *ThreadSafeComponent) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[key] = value
}
```

### Message Bus Usage

1. **Always unsubscribe** when an entity is shutting down:
   ```go
   func (e *Entity) Shutdown() error {
       return e.messageBus.Unsubscribe(e.id)
   }
   ```

2. **Handle message processing errors** properly:
   ```go
   messageBus.Subscribe(id, func(msg messaging.Message) error {
       // Catch panics
       defer func() {
           if r := recover(); r != nil {
               log.Printf("Recovered from panic in message handler: %v", r)
           }
       }()
       
       // Process message
       // ...
       
       return nil // or return error if something went wrong
   })
   ```

3. **Avoid slow operations** in message handlers:
   ```go
   // Good: Process in a separate goroutine
   messageBus.Subscribe(id, func(msg messaging.Message) error {
       go processMessage(msg) // Long-running operation in a separate goroutine
       return nil
   })
   
   // Bad: Blocking the message bus
   messageBus.Subscribe(id, func(msg messaging.Message) error {
       time.Sleep(5 * time.Second) // Don't do this!
       return nil
   })
   ```

### Tracer Usage

1. **Always close tracers** to ensure all data is flushed:
   ```go
   tracer, err := tracing.CreateFileTracer(...)
   if err != nil {
       // handle error
   }
   defer tracer.Close() // Ensure tracer is closed
   ```

2. **Use appropriate log levels**:
   ```go
   // Use Debug for developer information
   tracer.Debug("Processing message: %s", messageID)
   
   // Use Info for operational information
   tracer.Info("Server started on port %d", port)
   
   // Use Warning for potential issues
   tracer.Warning("Disk space low: %d%% used", diskUsage)
   
   // Use Error for actual errors
   tracer.Error("Failed to process request: %v", err)
   ```

## Testing

### Testing with Mock Tracers

```go
func TestMyComponent(t *testing.T) {
    // Create a mock tracer
    mockTracer := tracing.NewNoopTracer()
    
    // Create component with mock tracer
    component := NewMyComponent(mockTracer)
    
    // Test component
    // ...
}
```

### Testing with Mock Message Bus

```go
type MockMessageBus struct {
    // Implement MessageBus interface for testing
}

func (m *MockMessageBus) Publish(msg messaging.Message) error {
    // Record published messages for assertions
    return nil
}

// Implement other required methods...

func TestEntityWithMockBus(t *testing.T) {
    mockBus := &MockMessageBus{}
    entity := NewMyEntity(mockBus)
    
    // Test entity behavior
    // ...
}
```

## Performance Considerations

1. **Buffer size for file tracers**: Adjust based on log volume and performance requirements
2. **Message handling in separate goroutines**: For long-running operations
3. **Appropriate log levels in production**: Use Info or higher in production environments
4. **Regular trace file rotation**: Implement log rotation for long-running services

## Additional Resources

- [Go Concurrency Patterns](https://blog.golang.org/pipelines)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
