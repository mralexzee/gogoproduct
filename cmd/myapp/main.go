package main

import (
	"context"
	"goproduct/internal/agent"
	"goproduct/internal/chat"
	"goproduct/internal/common"
	"goproduct/internal/entity"
	"goproduct/internal/llm"
	"goproduct/internal/logging"
	"goproduct/internal/memory"
	"goproduct/internal/messaging"
	"goproduct/internal/tracing"
	"time"
)

func main() {
	// Create context for the application
	ctx := context.Background()

	// Initialize both logging and tracing systems
	// Logging - for application events
	logger := logging.File("app.log", true) // Append to log file
	defer logger.Close()
	logging.Init(logger) // Set as default logger

	// Tracing - still used for detailed internal tracing
	tracer, err := tracing.CreateFileTracer(
		"./trace.log", // File path
		5*time.Second, // 5-second flush interval
		4096,          // 4KB buffer size
	)
	if err != nil {
		panic(err)
	}
	defer tracer.Close()

	// Log application start
	logging.Get().Info("Application started")
	tracer.Info("Application started")

	// Create in-memory message bus
	messageBus := messaging.NewMemoryMessageBus()
	tracer.Info("Message bus created")

	// Create runtime context with message bus
	runtime, err := common.NewRuntimeContext(common.RuntimeOptions{
		MessageBus: messageBus,
	})
	if err != nil {
		panic(err)
	}
	tracer.Info("Runtime context created")

	// Create LLM for LM Studio
	llm, err := llm.NewLMStudioLLM("http://localhost:1234/v1",
		llm.WithLMStudioModel("gemma-3-4b-it"),
		llm.WithLMStudioTemperature(0.7),
		llm.WithLMStudioMaxTokens(4096),
		llm.WithLMStudioTimeout(60),
		llm.WithLMStudioTopP(0.9),
		llm.WithLMStudioPresencePenalty(0.0),
		llm.WithLMStudioFrequencyPenalty(0.0),
	)
	if err != nil {
		panic(err)
	}
	tracer.Info("LLM created")

	// Create file memory store
	store, err := memory.NewFileMemoryStore("./memories.json")
	if err != nil {
		panic(err)
	}
	err = store.Open()
	if err != nil {
		panic(err)
	}
	defer store.Close()

	// Set memory store in runtime context
	runtime.SetMemory(store)
	tracer.Info("Memory store created and added to runtime context")

	store.AddRecord(memory.MemoryRecord{
		ID:          "1",
		Category:    memory.CategoryFact,
		ContentType: memory.ContentTypeText,
		Content:     []byte("This is a fact."),
		Importance:  memory.ImportanceHigh,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		SourceID:    "1",
		SourceType:  "chat",
		OwnerID:     "1",
		OwnerType:   "agent",
		SubjectIDs:  []string{"1"},
		SubjectType: "agent",
		Tags:        []string{"fact"},
		References:  []memory.Reference{},
		Metadata:    map[string]string{"source": "chat"},
	})
	err = store.Flush()
	if err != nil {
		panic(err)
	}

	// Create an agent persona
	persona := agent.Persona{
		Name: "Andy",
		Role: "Assistant",
		Type: "Text",
		SystemPrompt: `
You are an AI Product Owner for a software company that creates websites, HTTP REST services, Android apps, iOS apps, Windows apps, and macOS apps. The CEO is your primary human stakeholder.

# Responsibilities
- Gather and clarify requirements.
- Create and maintain product roadmaps, plans, and specifications.
- Coordinate across departments to ensure alignment.
- Prioritize the product backlog for maximum value.
- Provide strategic, outcome-focused guidance.

# Communication & Tone
- Greet casually (e.g., “Hey, what’s up?”).
- Keep replies short, warm, and to the point—like a helpful teammate.
- Avoid formal or overly detailed language.
- Never use vulgar or insulting language.
- Stay friendly, polite, and adaptive to feedback.

# Scope & Limitations
- Focus strictly on product development topics.
- Politely decline requests unrelated to product development (e.g., weather updates, math solutions, personal opinions unrelated to the product).
- When greeted informally (e.g., “Hello” or “Hey”), respond in a brief, friendly way. If the user asks about or references product matters, respond with strategic, product-focused guidance.
`,
		LanguageModels: agent.LanguageModels{
			Default: llm,
		},
	}

	// Create the agent with the persona
	agentInstance := agent.NewAgent(persona)
	tracer.Info("Agent created")

	// Create entity adapters
	// 1. ProductAgentEntity for the agent
	productAgent := entity.NewProductAgentEntity(agentInstance, messageBus)
	tracer.Info("Product agent entity created: %s (%s)", productAgent.Name(), productAgent.ID())

	// 2. CliHumanEntity for the user
	humanaEntity := entity.NewCliHumanEntity("User", messageBus)
	tracer.Info("Human entity created: %s (%s)", humanaEntity.Name(), humanaEntity.ID())

	// Start the agent entity
	err = productAgent.Start(ctx)
	if err != nil {
		tracer.Error("Failed to start product agent: %v", err)
		panic(err)
	}
	tracer.Info("Product agent started")

	// Create enhanced chat interface
	chatInterface := chat.NewEnhancedChat(
		humanaEntity,
		productAgent,
		messageBus,
		tracer,
	)
	tracer.Info("Enhanced chat interface created")

	// Start the chat interface
	chatInterface.Start()
}
